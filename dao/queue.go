package dao

import (
	"database/sql"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/zql"
)

func NewQueue(db *zql.SynchronizedDB) (d Queue, err error) {
	d.db = db
	d.init()
	return
}

type Queue struct {
	db *zql.SynchronizedDB
}

func (d Queue) init() (err error) {
	_, err = d.db.Exec(`
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
	`)
	return
}

func (d Queue) QueueOperater(op model.Operater) (err error) {
	b, err := model.SerializeOp(op)
	if err != nil {
		return
	}
	logger.Debug("queueing operater", "kind", op.Kind())
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT OR IGNORE INTO suite_queue(name, open) VALUES (@suite, 0);
		INSERT INTO operation_queue(suite, op, unqueued, block, exitCode) 
			VALUES (@suite, @opBlob, 0, @block, NULL);
		`, sql.Named("suite", op.Suite()), sql.Named("opBlob", b), sql.Named("block", op.Block()))
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

	op.SetId(uint16(id))
	logger.Debug("queued operater", "id", id, "suite", op.Suite())

	return
}

func (d Queue) IsOperationsDone(op model.Operater) (done bool, exitCode int16, err error) {
	row := d.db.QueryRow(`
		SELECT exitCode 
		FROM operation_queue 
		WHERE id = @opId AND exitCode IS NOT NULL;
	`, sql.Named("opId", op.Id()))
	err = row.Scan(&exitCode)
	if err == sql.ErrNoRows {
		err = nil
		return
	} else if err != nil {
		return
	}
	done = true
	return
}

func (d Queue) isOperationsDone0(op model.Operater) (done bool, exitCode int16, err error) {
	row := d.db.QueryRow(`
		SELECT count(*) = 1, exitCode 
		FROM operation_queue 
		WHERE id = @opId AND exitCode IS NOT NULL;
	`, sql.Named("opId", op.Id()))
	err = row.Scan(&done, &exitCode)
	return
}

func (d Queue) QueuedSuites() (queued []string, err error) {
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

func (d Queue) OpenedNotBlockingSuites() (opened []string, err error) {
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

func (d Queue) Done(op model.Operater) (err error) {
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
	//logger.Warn("op done", "suite", suite, "op", op)
	return
}

func (d Queue) CloseSuite(suite string) (err error) {
	_, err = d.db.Exec(`UPDATE suite_queue SET open = 0 WHERE name = @suite;`, sql.Named("suite", suite))
	return
}

func (d Queue) QueuedOperationsCount() (count int, err error) {
	row := d.db.QueryRow("SELECT count(*) FROM operation_queue WHERE unqueued = 0;")
	err = row.Scan(&count)
	return
}

func (d Queue) QueuedOperationsCountBySuite(suite string, tx *zql.SynchronizedTx) (count int, err error) {
	var qr zql.SqlQuerier
	qr = d.db
	if tx != nil {
		qr = tx
	}
	row := qr.QueryRow("SELECT count(*) FROM operation_queue WHERE suite = ? and unqueued = 0;", suite)
	err = row.Scan(&count)
	return
}

func (d Queue) GlobalOperationsCount() (count int, err error) {
	row := d.db.QueryRow("SELECT count(*) FROM operation_queue;")
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

	op, err = model.DeserializeOp(b)
	op.SetId(opId)
	return
}

func (d Queue) UnqueueOperater() (op model.Operater, err error) {
	// 1- Elect suite : first already open not blocking suite
	// 2- Get next operation
	// 3- Record blocking state
	// 4- Remove operation from queue
	// 5- Remove suite if queue empty

	// Get first opened not blocking suite
	var electedSuite string
	openedNotBlockingSuites, err := d.OpenedNotBlockingSuites()
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
	op, err = d.NextQueuedOperation(electedSuite, tx)
	if err != nil {
		return
	}
	if op == nil {
		//logger.Warn("unqueueOperater() no operation found")
		return
	}
	opId := op.Id()

	logger.Debug("unqueueOperater()", "electedSuite", electedSuite, "opId", opId)

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
