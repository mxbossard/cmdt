package dao

import (
	"mby.fr/cmdtest/model"
	"mby.fr/utils/zql"
)

func NewTest(db *zql.SynchronizedDB) (d Test, err error) {
	d.db = db
	d.init()
	return
}

type Test struct {
	db *zql.SynchronizedDB
}

func (d Test) init() (err error) {
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suite TEXT NOT NULL,
			seq INTEGER NOT NULL,
			FOREIGN KEY(suite) REFERENCES suite(name)
		);
	`)
	return
}

func (d Test) SaveTest(suite string, seq int, cfg model.Config) (err error) {
	// TODO
	return
}

func (d Test) ClearSuite(suite string) (err error) {
	// TODO
	return
}
