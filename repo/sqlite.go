package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
	"mby.fr/utils/collections"
	_ "modernc.org/sqlite"
)

const DbFileName = "cmdt.sqlite"

func dbOpen(dirpath string) (db *sql.DB, err error) {
	file := filepath.Join(dirpath, DbFileName)

	//var initNeeded bool
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		//initNeeded = true
		defer dbInit(db)
	} else if err != nil {
		return
	}

	db, err = sql.Open("sqlite", file)
	if err != nil {
		return
	}
	/*
		if initNeeded {
			dbInit(db)
		}
	*/

	return
}

func dbInit(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS operation_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		suite TEXT NOT NULL,
		op TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS suite_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE
	); 
	CREATE TABLE IF NOT EXISTS opened_suite (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		blocking TEXT
	); 
	`)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

func queueOperater(db *sql.DB, op Operater) (err error) {
	var content []byte
	sop := buildSerializedOp(op)
	content, err = yaml.Marshal(sop)
	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`
	INSERT INTO operation_queue(suite, op) VALUES (%[1]s, %[2]s);
	INSERT INTO suite_queue(name) VALUES(%[1]s)
	`, op.Suite(), string(content))
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

func queuedOperations(db *sql.DB) (ops map[string][]Operater, err error) {

}

func queuedSuites(db *sql.DB) (queued []string, err error) {
	rows, err := db.Query("SELECT * FROM suite_queue;")
	if err != nil {
		return
	}

	for rows.Next() {
		var col string
		if err = rows.Scan(&col); err != nil {
			return
		}
		queued = append(queued, col)
	}
	return
}

func openedSuites(db *sql.DB) (opened []string, err error) {
	rows, err := db.Query("SELECT * FROM opened_suite;")
	if err != nil {
		return
	}

	for rows.Next() {
		var col string
		if err = rows.Scan(&col); err != nil {
			return
		}
		opened = append(opened, col)
	}
	return
}

type dbRepo struct {
	db         *sql.DB
	lastUpdate time.Time
}

func (r dbRepo) Queue(op Operater) {
	testSuite := op.Suite()
	logger.Debug("Queue()", "testSuite", testSuite, "operation", op)

	err := queueOperater(r.db, op)
	if err != nil {
		panic(err)
	}
}

func (r dbRepo) Unqueue() (ok bool, op Operater) {
	err := r.Update()
	if err != nil {
		panic(err)
	}
	if len(r.QueuedSuites) == 0 {
		return
	}
	logger.Debug("Unqueue()", "QueuedSuites", r.QueuedSuites, "OpenedSuites", r.OpenedSuites)
	var electedSuite string
	for _, suite := range r.OpenedSuites {
		// Elect first open not blocked queue
		q := r.Queues[suite]
		if q.Blocking != nil {
			// blocked queue => cannot elect it
			continue
		}
		electedSuite = r.QueuedSuites[0]
		break
	}

	if electedSuite == "" {
		// Elect first not opened queued suite
		for _, suite := range r.QueuedSuites {
			if !collections.Contains(&r.OpenedSuites, suite) {
				electedSuite = suite
				break
			}
		}
	}

	if electedSuite != "" {
		// open the queue if not done already
		if !collections.Contains(&r.OpenedSuites, electedSuite) {
			r.OpenedSuites = append(r.OpenedSuites, electedSuite)
		}
	} else {
		// no queue available to unqueue
		return
	}

	q := r.Queues[electedSuite]
	if q.Blocking != nil {
		err = fmt.Errorf("q %s was elected but is blocked", electedSuite)
		panic(err)
	}
	size := len(q.Operations)
	logger.Debug("Unqueue()", "electedSuite", electedSuite, "size", size)

	if size > 0 {
		// Unqueue operation
		ok = true
		sop := q.Operations[0]
		op = deserializeOp(sop)
		q.Operations = q.Operations[1:]
		if op.Block() {
			q.Blocking = &sop
		}
		r.Queues[electedSuite] = q
	}

	if len(q.Operations) == 0 {
		logger.Debug("Unqueue() clearing QueuedSuites")
		// Empty queue => remove it
		if len(r.QueuedSuites) == 1 {
			r.QueuedSuites = []string{}
		} else {
			for p, s := range r.QueuedSuites {
				if s == electedSuite {
					r.QueuedSuites = append(r.QueuedSuites[:p], r.QueuedSuites[p+1:]...)
					break
				}
			}
		}
		delete(r.Queues, electedSuite)
	}

	logger.Warn("Unqueue()", "ok", ok, "ok2", op != nil, "opened", r.OpenedSuites, "electedSuite", electedSuite, "remaining", len(q.Operations), "blocked", q.Blocking != nil)

	return
}

func (r dbRepo) Unblock(op Operater) {
	if op == nil {
		return
	}
	suite := op.Suite()
	q := r.Queues[suite]
	if q.Blocking != nil {
		blocking := deserializeOp(*q.Blocking)
		if blocking.Kind() == op.Kind() && blocking.Seq() == op.Seq() {
			q.Blocking = nil
			logger.Warn("Unblock", "suite", suite)
		}
	}
}

func (r dbRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		err := r.Update()
		if err != nil {
			panic(err)
		}
		if q, ok := r.Queues[testSuite]; ok {
			logger.Warn("WaitEmptyQueue()", "q", q)
			if len(q.Operations) == 0 {
				// Queue is empty
				return
			}
		} else {
			// Queue does not exists
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	err := errors.New("WaitEmptyQueue() timed out")
	panic(err)
}

func (r dbRepo) WaitAllEmpty(timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		err := r.Update()
		if err != nil {
			panic(err)
		}
		if len(r.Queues) == 0 {
			// No Queue anymore
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	err := errors.New("WaitAllEmpty() timed out")
	panic(err)
}
