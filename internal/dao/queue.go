package dao

import (
	"database/sql"
	"errors"
	"os"

	"cmdt/internal/model"

	"github.com/mxbossard/utilz/zql"
	"github.com/mxbossard/utilz/zqlite"
)

func NewQueue(db *zqlite.SynchronizedDB, init bool) (d Queue, err error) {
	d.db = db
	if init {
		err = d.init()
	}
	return
}

type Queue struct {
	db *zqlite.SynchronizedDB
}

func (d Queue) init() (err error) {
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS suite_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			open INTEGER NOT NULL,
			blocking INTEGER
		);
		CREATE INDEX IF NOT EXISTS suite_queue_name ON suite_queue(name);
		CREATE INDEX IF NOT EXISTS suite_queue_open ON suite_queue(open);
		CREATE INDEX IF NOT EXISTS suite_queue_open_blocking ON suite_queue(open, blocking);
		
		CREATE TABLE IF NOT EXISTS operation_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suite TEXT NOT NULL,
			op BLOB NOT NULL,
			unqueued INTEGER NOT NULL,
			exitCode INTEGER,
			error TEXT,
			block INTEGER,
			FOREIGN KEY(suite) REFERENCES suite_queue(name)
		);
		CREATE INDEX IF NOT EXISTS operation_queue_id ON operation_queue(id);
		CREATE INDEX IF NOT EXISTS operation_queue_suite ON operation_queue(suite);
		CREATE INDEX IF NOT EXISTS operation_queue_unqueued ON operation_queue(unqueued);
		CREATE INDEX IF NOT EXISTS operation_queue_suite_unqueued ON operation_queue(suite, unqueued);
		CREATE INDEX IF NOT EXISTS operation_queue_id_exitCode ON operation_queue(id, exitCode);
		CREATE INDEX IF NOT EXISTS operation_queue_id_unqueued ON operation_queue(id, unqueued);
	`)
	return
}

func (d Queue) QueueOperater(op model.Operater) (err error) {
	perf := logger.PerfTimer("kind", op.Kind())
	defer perf.End("op", op, "err", err)

	b, err := model.SerializeOp(op)
	if err != nil {
		return
	}

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT OR IGNORE INTO suite_queue(name, open) VALUES (@suite, 0);
		INSERT INTO operation_queue(suite, op, unqueued, block, exitCode, error) 
			VALUES (@suite, @opBlob, 0, @block, NULL, NULL);
		`, sql.Named("suite", op.Suite()), sql.Named("opBlob", b), sql.Named("block", op.Block())) // OR IGNORE
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		return
	}

	err = tx.Commit()
	if err != nil {
		return
	}

	op.SetId(uint(id))

	return
}

func (d Queue) IsOperationsDone(op model.Operater) (done bool, exitCode int16, opErr, err error) {
	exitCode = -1
	var errMsg string
	row := d.db.QueryRow(`
		SELECT q.exitCode, COALESCE(q.error, '')
		FROM operation_queue q
		WHERE q.id = @opId AND q.exitCode IS NOT NULL;
	`, sql.Named("opId", op.Id()))
	err = row.Scan(&exitCode, &errMsg)
	if err == sql.ErrNoRows {
		err = nil
		return
	} else if err != nil {
		return
	}

	if errMsg != "" {
		opErr = errors.New(errMsg)
	}
	done = true
	return
}

func (d Queue) QueuedSuites0() (queued []string, err error) {
	rows, err := d.db.Query(`
		SELECT s.name 
		FROM suite_queue s 
		WHERE s.open = 0 OR (s.open > 0 AND s.open <> @pid)
		ORDER BY s.id;
	`, sql.Named("pid", os.Getpid()))
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

func (d Queue) OpenedNotBlockingSuites() (opened []string, err error) {
	perf := logger.PerfTimer()
	defer perf.End("opened", opened)

	rows, err := d.db.Query(`
		SELECT s.name 
		FROM suite_queue s
		WHERE s.open > 0 AND (s.blocking IS NULL OR s.open <> @pid)
		ORDER BY s.id;
	`, sql.Named("pid", os.Getpid()))
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

func (d Queue) electSuite(tx *zql.SynchronizedTx) (elected string, err error) {
	perf := logger.PerfTimer()
	defer perf.End("elected", elected)

	// Elect suite with followinf rules
	// 1- Only suite with already queued operations
	// 2- Suite which are not blocked by current daemon
	// 3- Priorize already opened suite
	// 4- Priorize oldest suite
	row := tx.QueryRow(`
		SELECT s.name 
		FROM suite_queue s LEFT JOIN operation_queue o ON s.name = o.suite
		WHERE o.exitCode IS NULL AND (s.blocking IS NULL OR o.unqueued <> :pid)
		ORDER BY s.open DESC, s.id ASC
		LIMIT 1
	`, sql.Named("pid", os.Getpid()))

	err = row.Scan(&elected)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// Requeue not done operation
func (d Queue) NotDone(op model.Operater) (err error) {
	perf := logger.PerfTimer()
	defer perf.End()

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
			UPDATE operation_queue 
			SET unqueued = 0, exitCode = NULL, error = NULL 
			WHERE id = @opId;
			
			UPDATE suite_queue 
			SET blocking = NULL
			WHERE name = @suite AND blocking = @opId;
		`, sql.Named("suite", op.Suite()), sql.Named("opId", op.Id()))

	return
}

func (d Queue) Done(op model.Operater) (err error) {
	perf := logger.PerfTimer()
	defer perf.End()

	// 1- Flag operation done
	// 2- Flag suite done or Remove suite if no operation remaining

	suite := op.Suite()
	//logger.Warn("doning op ...", "suite", suite, "op", op)
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	count, err := d.QueuedOperationsCountBySuite(suite, tx)
	if err != nil {
		return
	}

	var errMsg *string
	if op.Err() != nil {
		msg := op.Err().Error()
		errMsg = &msg
	}
	_, err = tx.Exec(`UPDATE operation_queue SET unqueued = -1, exitCode = ?, error = ? WHERE id = ?;`, op.ExitCode(), errMsg, op.Id())
	if err != nil {
		return
	}

	if count == 0 {
		// Remove suite
		logger.Debug("deleting suite_queue", "name", suite)
		_, err = tx.Exec(`DELETE FROM suite_queue WHERE name = ?;`, suite)
	} else {
		// Unblock suite
		logger.Debug("unblocking suite_queue", "suite", suite)
		_, err = tx.Exec(`UPDATE suite_queue SET blocking = NULL WHERE name = ?;`, suite)
	}
	if err != nil {
		return
	}

	err = tx.Commit()
	//logger.Warn("op done", "suite", suite, "op", op)
	return
}

func (d Queue) CountGlobalNotDoneBefore(op model.Operater) (count int, err error) {
	row := d.db.QueryRow(`
		SELECT count(*) 
		FROM operation_queue q
		WHERE q.id < @id AND q.exitCode IS NULL;
	`, sql.Named("id", op.Id()))
	err = row.Scan(&count)
	return
}

func (d Queue) CountSuiteNotDoneBefore(op model.Operater) (count int, err error) {
	suite := op.Suite()
	if suite == "" {

	}
	row := d.db.QueryRow(`
		SELECT count(*) 
		FROM operation_queue q
		WHERE q.id < @id AND q.suite = @suite AND q.exitCode IS NULL;
	`, sql.Named("suite", suite), sql.Named("id", op.Id()))
	err = row.Scan(&count)
	return
}

func (d Queue) CloseSuite0(suite string) (err error) {
	_, err = d.db.Exec(`UPDATE suite_queue SET open = 0 WHERE name = @suite;`, sql.Named("suite", suite))
	return
}

func (d Queue) DeleteQueuesOfSuite(suite string) (err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM suite_queue
		WHERE name = @suite;
		DELETE FROM operation_queue
		WHERE suite = @suite;
	`, sql.Named("suite", suite))
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

func (d Queue) QueuedOperationsCount() (count int, err error) {
	row := d.db.QueryRow(`
		SELECT count(*) 
		FROM operation_queue q
		WHERE q.unqueued > -1;
	`) //  AND q.unqueued <> @pid  // sql.Named("pid", os.Getpid())
	err = row.Scan(&count)
	return
}

func (d Queue) QueuedOperationsCountBySuite(suite string, tx *zql.SynchronizedTx) (count int, err error) {
	var qr zql.SqlQuerier
	qr = d.db
	if tx != nil {
		qr = tx
	}
	row := qr.QueryRow(`
		SELECT count(*) 
		FROM operation_queue q
		WHERE q.suite = ? and q.unqueued > -1;
	`, suite) //  AND q.unqueued <> @pid  // , sql.Named("pid", os.Getpid())
	err = row.Scan(&count)
	return
}

func (d Queue) GlobalOperationsCount() (count int, err error) {
	row := d.db.QueryRow(`
		SELECT count(*) 
		FROM operation_queue q
	;`)
	err = row.Scan(&count)
	return
}

func (d Queue) NextQueuedOperation(suite string, tx *zql.SynchronizedTx) (op model.Operater, err error) {
	var qr zql.SqlQuerier
	qr = d.db
	if tx != nil {
		qr = tx
	}
	row := qr.QueryRow(`
		SELECT q.id, q.op 
		FROM operation_queue q
		WHERE q.suite = @suite and (q.unqueued > -1 AND q.unqueued <> @pid)
		ORDER BY q.id 
		LIMIT 1;
	`, sql.Named("suite", suite), sql.Named("pid", os.Getpid()))
	var b []byte
	var opId uint
	err = row.Scan(&opId, &b)
	if err == sql.ErrNoRows {
		// No operation queued
		err = nil
		//logger.Warn("no operation found")
		return
	} else if err != nil {
		return
	}

	op, err = model.DeserializeOp(b)
	op.SetId(opId)
	return
}

func (d Queue) UnqueueOperater() (op model.Operater, err error) {
	perf := logger.PerfTimer()
	defer perf.End("op", op, "err", err)

	// 1- Elect suite : first already open not blocking suite
	// 2- Get next operation
	// 3- Record blocking state
	// 4- Remove operation from queue
	// 5- Remove suite if queue empty

	// Get first opened not blocking suite
	var electedSuite string

	/*
		openedNotBlockingSuites, err := d.OpenedNotBlockingSuites()
		if err != nil {
			return
		}
		if len(openedNotBlockingSuites) > 0 {
			electedSuite = openedNotBlockingSuites[0]
		}

		logger.Trace("UnqueueOperater() 1", "electedSuite", electedSuite)

		if electedSuite == "" {
			// Select first closed suite
			row := d.db.QueryRow(`
				SELECT s.name
				FROM suite_queue s
				WHERE s.open = 0 OR (s.open > 0 AND s.open <> @pid)
				ORDER BY s.id
				LIMIT 1;
			`, sql.Named("pid", os.Getpid()))
			err = row.Scan(&electedSuite)
			if err == sql.ErrNoRows {
				logger.Debug("no closed suite_queue found")
				err = nil
			} else if err != nil {
				return
			}

			//fmt.Printf("\n<<>> not already opened electedSuite: %s\n", electedSuite)
			if electedSuite == "" {
				// No suite found
				return
			}
		}
	*/

	//fmt.Printf("\n<<>> electedSuite: %s\n", electedSuite)
	logger.Trace("UnqueueOperater() 2", "electedSuite", electedSuite)

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	electedSuite, err = d.electSuite(tx)
	if err != nil {
		return
	}

	if electedSuite == "" {
		// No suite found
		return
	}

	// Get next operation
	op, err = d.NextQueuedOperation(electedSuite, tx)
	if err != nil {
		return
	}
	if op == nil {
		logger.Trace("UnqueueOperater() no operation found")
		return
	}
	opId := op.Id()

	logger.Debug("UnqueueOperater()", "electedSuite", electedSuite, "opId", opId)

	// Open this suite & Record blocking state
	if op.Block() {
		_, err = tx.Exec(`
			UPDATE suite_queue SET open = @pid, blocking = @opId
			WHERE name = @suite;
	`, sql.Named("pid", os.Getpid()), sql.Named("suite", electedSuite), sql.Named("opId", op.Id()))
	} else {
		_, err = tx.Exec(`
			UPDATE suite_queue SET open = @pid, blocking = NULL
			WHERE name = @suite;
	`, sql.Named("pid", os.Getpid()), sql.Named("suite", electedSuite))
	}

	if err != nil {
		return
	}

	// Remove operation
	_, err = tx.Exec(`UPDATE operation_queue SET unqueued = @pid WHERE id = @id;`,
		sql.Named("pid", os.Getpid()), sql.Named("id", opId))
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}
