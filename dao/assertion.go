package dao

import "mby.fr/utils/zql"

func NewAssertion(db *zql.SynchronizedDB) (d Assertion, err error) {
	d.db = db
	d.init()
	return
}

type Assertion struct {
	db *zql.SynchronizedDB
}

func (d Assertion) init() (err error) {
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS assertion (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			test INTEGER NOT NULL,
			seq INTEGER NOT NULL,

			FOREIGN KEY(test) REFERENCES test(name)
		);
	`)
	return
}
