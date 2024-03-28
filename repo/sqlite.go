package repo

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
	_ "modernc.org/sqlite"
)

const (
	DbFileName  = "cmdt.sqlite"
	BusyTimeout = 5 * time.Second
)

func dbOpen(dirpath string) (db *OneWriterDB, err error) {
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

	//db, err = sql.Open("sqlite", file+"?_busy_timeout=5000")
	db, err = OpenSqliteOneWriterDB(file, "")
	if err != nil {
		return
	}

	db.SetMaxOpenConns(5)

	return
}

func dbInit(db *OneWriterDB) (err error) {
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS suite_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			open INTEGER NOT NULL,
			blocking INTEGER
		);

		CREATE TABLE IF NOT EXISTS operation_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suite TEXT NOT NULL,
			op BLOB NOT NULL,
			unqueued INTEGER NOT NULL,
			exitCode INTEGER,
			block INTEGER,
			FOREIGN KEY(suite) REFERENCES suite_queue(name)
		);

		CREATE TABLE IF NOT EXISTS opened_suite (
			name TEXT PRIMARY KEY,
			blocking INTEGER,
			FOREIGN KEY(name) REFERENCES suite_queue(name)
		);
	`)
	return
}

type SqlQuerier interface {
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type OneWriterDB struct {
	*sql.DB
	fileLock    *flock.Flock
	busyTimeout time.Duration
}

func (d OneWriterDB) lock() (err error) {
	lockCtx, cancel := context.WithTimeout(context.Background(), d.busyTimeout)
	defer cancel()
	locked, err := d.fileLock.TryLockContext(lockCtx, time.Millisecond)
	if err != nil {
		return
	}
	if !locked {
		err = errors.New("unable to acquire DB lock")
	}
	return
}

func (d OneWriterDB) unlock() (err error) {
	if d.fileLock != nil {
		err = d.fileLock.Unlock()
	}
	return
}

func (d OneWriterDB) Exec(query string, args ...any) (sql.Result, error) {
	d.lock()
	defer d.unlock()
	return d.DB.Exec(query, args...)
}

func (d OneWriterDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.lock()
	defer d.unlock()
	return d.DB.ExecContext(ctx, query, args...)
}

func (d OneWriterDB) Query(query string, args ...any) (*sql.Rows, error) {
	d.lock()
	defer d.unlock()
	return d.DB.Query(query, args...)
}

func (d OneWriterDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	d.lock()
	defer d.unlock()
	return d.DB.QueryContext(ctx, query, args...)
}

func (d OneWriterDB) QueryRow(query string, args ...any) *sql.Row {
	d.lock()
	defer d.unlock()
	return d.DB.QueryRow(query, args...)
}

func (d OneWriterDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	d.lock()
	defer d.unlock()
	return d.DB.QueryRowContext(ctx, query, args...)
}

func (d OneWriterDB) Begin() (*OneWriterTx, error) {
	d.lock()
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &OneWriterTx{Tx: tx, db: &d}, nil
}

func (d OneWriterDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*OneWriterTx, error) {
	d.lock()
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &OneWriterTx{Tx: tx, db: &d}, nil
}

type OneWriterTx struct {
	*sql.Tx
	db *OneWriterDB
}

func (t OneWriterTx) Commit() error {
	defer t.db.unlock()
	return t.Tx.Commit()
}

func (t OneWriterTx) Rollback() error {
	defer t.db.unlock()
	return t.Tx.Rollback()
}

func OpenSqliteOneWriterDB(backingFile, opts string) (*OneWriterDB, error) {
	dataSourceName := backingFile
	if opts != "" {
		dataSourceName += "?" + opts
	}
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	fileLock := flock.New(backingFile)
	wrapper := OneWriterDB{
		DB:          db,
		fileLock:    fileLock,
		busyTimeout: BusyTimeout,
	}
	return &wrapper, err
}

type queueDao struct {
	db *OneWriterDB
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
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT OR IGNORE INTO suite_queue(name, open) VALUES (@suite, 0);
		INSERT INTO operation_queue(suite, op, unqueued, block) 
			VALUES (@suite, @opBlob, 0, @block);
		`, sql.Named("suite", op.Suite()), sql.Named("opBlob", b), sql.Named("block", op.Block()))
	if err != nil {
		return
	}
	err = tx.Commit()
	if err != nil {
		return
	}

	// queuedSuites, err := d.queuedSuites()
	// if err != nil {
	// 	return
	// }
	// logger.Warn("queueOperater()", "queuedSuites", queuedSuites)

	return
}

func (d queueDao) queuedSuites() (queued []string, err error) {
	rows, err := d.db.Query("SELECT name FROM suite_queue WHERE open = 0 ORDER BY id;")
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

func (d queueDao) done(op Operater) (err error) {
	// 1- Flag operation done
	// 2- Flag suite done or Remove suite if no operation remaining

	suite := op.Suite()

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	count, err := d.queuedOperationsCountBySuite(suite, tx)
	if err != nil {
		return
	}

	_, err = tx.Exec(`UPDATE operation_queue SET exitCode = ? WHERE id = ?;`, op.ExitCode(), op.Id())
	if err != nil {
		return
	}

	if count == 0 {
		// Remove suite
		_, err = tx.Exec(`DELETE FROM suite_queue WHERE name = ?;`, suite)
	} else {
		// Unblock suite
		_, err = tx.Exec(`UPDATE suite_queue SET blocking = NULL WHERE name = ?;`, suite)
	}
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

func (d queueDao) closeSuite(suite string) (err error) {
	_, err = d.db.Exec(`UPDATE suite_queue SET open = 0 WHERE name = @suite;`, sql.Named("suite", suite))
	return
}

func (d queueDao) queuedOperationsCount() (count int, err error) {
	row := d.db.QueryRow("SELECT count(*) FROM operation_queue WHERE unqueued = 0;")
	err = row.Scan(&count)
	return
}

func (d queueDao) queuedOperationsCountBySuite(suite string, tx *OneWriterTx) (count int, err error) {
	var qr SqlQuerier
	qr = d.db
	if tx != nil {
		qr = tx
	}
	row := qr.QueryRow("SELECT count(*) FROM operation_queue WHERE suite = ? and unqueued = 0;", suite)
	err = row.Scan(&count)
	return
}

func (d queueDao) globalOperationsCount() (count int, err error) {
	row := d.db.QueryRow("SELECT count(*) FROM operation_queue;")
	err = row.Scan(&count)
	return
}

func (d queueDao) isOperationsDone(op Operater) (done bool, exitCode int16, err error) {
	row := d.db.QueryRow(`
		SELECT count(*) = 1, exitCode 
		FROM operation_queue 
		WHERE id = @opId AND exitCode IS NOT NULL;
	`, sql.Named("opId", op.Id()))
	err = row.Scan(&done, &exitCode)
	return
}

func (d queueDao) nextQueuedOperation(suite string, tx *OneWriterTx) (op Operater, err error) {
	var qr SqlQuerier
	qr = d.db
	if tx != nil {
		qr = tx
	}
	row := qr.QueryRow(`
		SELECT id, op 
		FROM operation_queue 
		WHERE suite = @suite and unqueued = 0 
		ORDER BY id 
		LIMIT 1;
	`, sql.Named("suite", suite))
	var b []byte
	var opId uint16
	err = row.Scan(&opId, &b)
	if err == sql.ErrNoRows {
		// No operation queued
		err = nil
		//logger.Warn("no operation found")
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
		//logger.Warn("unqueueOperater()", "openedNotBlockingSuites", openedNotBlockingSuites)
	}

	//rollback := true
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	/*
		defer func() {
			var err2 error
			if rollback {
				err2 = tx.Rollback()
			} else {
				err2 = tx.Commit()
			}
			if err2 != nil {
				err = err2
			}
		}()
	*/

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
	}

	//logger.Warn("unqueueOperater()", "electedSuite", electedSuite)

	// Get next operation
	op, err = d.nextQueuedOperation(electedSuite, tx)
	if err != nil {
		return
	}
	if op == nil {
		//logger.Warn("unqueueOperater() no operation found")
		return
	}
	opId := op.Id()

	logger.Info("unqueueOperater()", "electedSuite", electedSuite, "opId", opId)

	// Open this suite & Record blocking state
	if op.Block() {
		_, err = tx.Exec(`
			UPDATE suite_queue SET open = 1, blocking = @opId
			WHERE name = @suite;
	`, sql.Named("suite", electedSuite), sql.Named("opId", op.Id()))
	} else {
		_, err = tx.Exec(`
			UPDATE suite_queue SET open = 1, blocking = NULL
			WHERE name = @suite;
	`, sql.Named("suite", electedSuite))
	}

	if err != nil {
		return
	}

	// Remove operation
	_, err = tx.Exec(`UPDATE operation_queue SET unqueued = 1 WHERE id = @id;`,
		sql.Named("id", opId))
	if err != nil {
		return
	}

	//rollback = false
	err = tx.Commit()
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
	//testSuite := op.Suite()
	//logger.Warn("Queue() adding", "testSuite", testSuite, "kind", op.Kind(), "seq", op.Seq())

	// start := time.Now()
	// for time.Since(start) < BusyTimeout {
	// BUSY retries
	err = r.dao.queueOperater(op)
	// 	if err == nil || !IsBusyError(err) {
	// 		break
	// 	}
	// }

	//logger.Warn("Queue() added", "testSuite", testSuite, "kind", op.Kind(), "seq", op.Seq())

	return
}

func (r dbRepo) unqueue() (ok bool, op Operater, err error) {
	queuedOperationsCount, err := r.dao.queuedOperationsCount()
	if err != nil {
		return
	}

	//globalOperationsCount, err := r.dao.globalOperationsCount()
	//if err != nil {
	//	return
	//}

	//logger.Warn("unqueue()", "globalOperationsCount", globalOperationsCount)

	if queuedOperationsCount == 0 {
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
	// start := time.Now()
	// for time.Since(start) < BusyTimeout {
	// BUSY retries
	ok, op, err = r.unqueue()
	// if err == nil || !IsBusyError(err) {
	// 	   break
	// }
	// 	logger.Warn("Retrying BUSY ...")
	// 	time.Sleep(time.Millisecond)
	// }

	//if ok {
	//	logger.Warn("Unqueue()", "kind", op.Kind(), "opId", op.Id())
	//}

	return
}

func (r dbRepo) Done(op Operater) (err error) {
	if op == nil {
		return
	}

	err = r.dao.done(op)
	//logger.Warn("Unblock() unblocked", "opId", op.Id())
	return
}

func (r dbRepo) WaitOperaterDone(op Operater, timeout time.Duration) (exitCode int16, err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var done bool
		done, exitCode, err = r.dao.isOperationsDone(op)
		if done || err != nil {
			// Operater not done
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	err = errors.New("WaitOperaterDone() timed out")
	return
}

func (r dbRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.dao.queuedOperationsCountBySuite(testSuite, nil)
		if err != nil {
			return
		}
		//logger.Warn("WaitEmptyQueue()", "testSuite", testSuite, "count", count)
		if count == 0 {
			// Queue is empty
			return
		}
		time.Sleep(10 * time.Millisecond)
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
		time.Sleep(10 * time.Millisecond)
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
