package repo

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	DbFileName  = "cmdt.sqlite"
	BusyTimeout = 5 * time.Second
)

func dbOpen(dirpath string) (db *sql.DB, err error) {
	file := filepath.Join(dirpath, DbFileName)

	//var initNeeded bool
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		//initNeeded = true
		defer func() {
			dbInit(db)
		}()
	} else if err != nil {
		return
	}

	db, err = sql.Open("sqlite", file+"?_busy_timeout=5000")
	if err != nil {
		return
	}

	db.SetMaxOpenConns(5)

	return
}

func dbInit(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS suite_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE,
			open INTEGER,
			blocking INTEGER
		);

		CREATE TABLE IF NOT EXISTS operation_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suite TEXT NOT NULL,
			op BLOB NOT NULL,
			block INTEGER,
			unqueued INTEGER,
			FOREIGN KEY(suite) REFERENCES suite_queue(name)
		);

		CREATE TABLE IF NOT EXISTS opened_suite (
			name TEXT PRIMARY KEY,
			blocking INTEGER,
			FOREIGN KEY(name) REFERENCES suite_queue(name)
		);
	`) // FOREIGN KEY(blocking) REFERENCES operation_queue(id)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

type queueDao struct {
	db *sql.DB
}

func (d queueDao) queueOperater(op Operater) (err error) {
	b, err := serializeOp(op)
	if err != nil {
		return
	}

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO suite_queue(name) VALUES (@suite);
		INSERT INTO operation_queue(suite, op, block, unqueued) VALUES (@suite, @opBlob, @block, 0);
		`, sql.Named("suite", op.Suite()), sql.Named("opBlob", b), sql.Named("block", op.Block()))
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			err = err2
		}
		return
	}
	err = tx.Commit()
	return
}

func (d queueDao) queuedSuites() (queued []string, err error) {
	rows, err := d.db.Query("SELECT * FROM suite_queue WHERE open = 1 ORDER BY id;")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var col string
		if err = rows.Scan(&col); err != nil {
			return
		}
		queued = append(queued, col)
	}
	return
}

func (d queueDao) openedNotBlockingSuites() (opened []string, err error) {
	rows, err := d.db.Query("SELECT name FROM suite_queue WHERE open = 1 AND blocking IS NULL ORDER BY id;")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var col string
		if err = rows.Scan(&col); err != nil {
			return
		}
		opened = append(opened, col)
	}
	return
}

func (d queueDao) unblockSuite(op Operater) (err error) {
	suite := op.Suite()
	_, err = d.db.Exec(`
		UPDATE suite_queue SET blocking = NULL 
		WHERE name = '@suite' and blocking = @id;
	`, sql.Named("suite", suite), sql.Named("id", op.Id()))
	return
}

func (d queueDao) closeSuite(suite string) (err error) {
	_, err = d.db.Exec(`UPDATE suite_queue SET open = 0 WHERE name = '@suite';`, sql.Named("suite", suite))
	return
}

func (d queueDao) queuedOperationsCount() (count int, err error) {
	row := d.db.QueryRow("SELECT count(*) FROM operation_queue WHERE unqueued = 0;")
	err = row.Scan(&count)
	return
}

func (d queueDao) queuedOperationsCountBySuite(suite string) (count int, err error) {
	row := d.db.QueryRow("SELECT count(*) FROM operation_queue WHERE suite = '?' and unqueued = 0;", suite)
	err = row.Scan(&count)
	return
}

func (d queueDao) nextQueuedOperation(suite string) (op Operater, err error) {
	row := d.db.QueryRow(`
		SELECT id, op 
		FROM operation_queue 
		WHERE suite = '?' and unqueued = 0 
		ORDER BY id 
		LIMIT 1;
	`, suite)
	var b []byte
	var opId uint16
	err = row.Scan(&opId, &b)
	if err == sql.ErrNoRows {
		// No operation queued
		err = nil
		return
	} else if err != nil {
		return
	}

	op, err = deserializeOp(b)
	op.SetId(opId)
	return
}

func (d queueDao) unqueueOperater() (op Operater, err error) {
	// 1- Elect suite : first already open not blocking suite
	// 2- Get next operation
	// 3- Record blocking state
	// 4- Remove operation from queue
	// 5- Remove suite if queue empty

	// Get first opened not blocking suite
	var electedSuite string
	openedNotBlockingSuites, err := d.openedNotBlockingSuites()
	if err != nil {
		return
	}
	if len(openedNotBlockingSuites) > 0 {
		electedSuite = openedNotBlockingSuites[0]
	}

	rollback := true
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		var err2 error
		if rollback {
			err2 = tx.Rollback()
			if err2 != nil {
				err = err2
			}
		} else {
			start := time.Now()
			for time.Since(start) < BusyTimeout {
				err2 = tx.Commit()
				if err2 == nil || !IsBusyError(err2) {
					break
				}
			}
		}
		if err2 != nil {
			err = err2
		}
	}()

	if electedSuite == "" {
		// Select first closed suite
		row := tx.QueryRow(`
			SELECT name 
			FROM suite_queue
			WHERE open = 0
			ORDER BY id
			LIMIT 1;
		`)
		err = row.Scan(&electedSuite)
		if err == sql.ErrNoRows {
			err = nil
		} else if err != nil {
			return
		}

		if electedSuite == "" {
			// No suite found
			return
		}

		// Open this suite
		_, err = tx.Exec(`
			UPDATE suite_queue SET open = 1
			WHERE name = @suite;
		`, sql.Named("suite", electedSuite))
		if err != nil {
			return
		}
	}

	// Get next operation
	op, err = d.nextQueuedOperation(electedSuite)
	if err != nil {
		return
	}
	opId := op.Id()

	logger.Info("unqueueOperater()", "electedSuite", electedSuite, "opId", opId)

	// Record blocking state
	if op.Block() {
		_, err = tx.Exec(`UPDATE suite_queue SET blocking = @opId WHERE name = '@suite';`,
			sql.Named("suite", electedSuite), sql.Named("opId", opId))
		if err != nil {
			return
		}
	}

	// Remove operation
	_, err = tx.Exec(`DELETE FROM operation_queue WHERE id = @id;`,
		sql.Named("id", opId))
	if err != nil {
		return
	}

	// Remove suite if queue empty
	_, err = tx.Exec(`DELETE FROM suite_queue WHERE name NOT IN (
		SELECT distinct(suite)
		FROM operation_queue
	);`)
	if err != nil {
		return
	}

	rollback = false
	return
}

func IsBusyError(err error) bool {
	return strings.Contains(err.Error(), "SQLITE_BUSY") || strings.Contains(err.Error(), "cannot start a transaction within a transaction")
}

type dbRepo struct {
	dao queueDao
	//lastUpdate time.Time
}

func (r dbRepo) Queue(op Operater) (err error) {
	testSuite := op.Suite()
	logger.Warn("Queue()", "testSuite", testSuite, "kind", op.Kind(), "seq", op.Seq())

	start := time.Now()
	for time.Since(start) < BusyTimeout {
		// BUSY retries
		err = r.dao.queueOperater(op)
		if err == nil || !IsBusyError(err) {
			break
		}
	}

	return
}

func (r dbRepo) unqueue() (ok bool, op Operater, err error) {
	var queuedOperationsCount int
	queuedOperationsCount, err = r.dao.queuedOperationsCount()
	if err != nil || queuedOperationsCount == 0 {
		return
	}

	op, err = r.dao.unqueueOperater()
	if err != nil || op == nil {
		return
	}

	ok = true
	return
}

func (r dbRepo) Unqueue() (ok bool, op Operater, err error) {
	start := time.Now()
	for time.Since(start) < BusyTimeout {
		// BUSY retries
		ok, op, err = r.unqueue()
		if err == nil || !IsBusyError(err) {
			break
		}
		logger.Warn("Retrying BUSY ...")
		time.Sleep(time.Millisecond)
	}

	if ok {
		logger.Warn("Unqueue()", "kind", op.Kind(), "seq", op.Seq())
	}

	return
}

func (r dbRepo) Unblock(op Operater) (err error) {
	if op == nil {
		return
	}

	err = r.dao.unblockSuite(op)
	return
}

func (r dbRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.dao.queuedOperationsCountBySuite(testSuite)
		if err != nil {
			return
		}
		if count == 0 {
			// Queue is empty
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	err = errors.New("WaitEmptyQueue() timed out")
	return
}

func (r dbRepo) WaitAllEmpty(timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.dao.queuedOperationsCount()
		if err != nil {
			return
		}
		if count == 0 {
			// No operation queued
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	err = errors.New("WaitAllEmpty() timed out")
	return
}

func newQueueDao(dir string) (d queueDao, err error) {
	d.db, err = dbOpen(dir)
	return
}

func newDbRepo(dir string) (d dbRepo, err error) {
	d.dao, err = newQueueDao(dir)
	return
}
