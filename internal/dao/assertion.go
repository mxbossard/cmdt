package dao

import "github.com/mxbossard/utilz/zql"

func NewAssertion(db *zql.SynchronizedDB, init bool) (d Assertion, err error) {
	d.db = db
	if init {
		err = d.init()
	}
	return
}

type Assertion struct {
	db *zql.SynchronizedDB
}

func (d Assertion) init() (err error) {
	_, err = d.db.Exec(`
		FOO
	`)
	return
}
