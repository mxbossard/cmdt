package dao

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	//_ "github.com/mattn/go-sqlite3"

	"mby.fr/utils/zlog"
	"mby.fr/utils/zql"
	"mby.fr/utils/zqlite"
)

const (
	DbFileName  = "cmdt.sqlite"
	BusyTimeout = 5 * time.Second
)

var (
	logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
)

func DbOpen(dirpath string) (db *zql.SynchronizedDB, err error) {
	file := filepath.Join(dirpath, DbFileName)

	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		/*
			defer func() {
				err2 := dbInit(db)
				if err2 != nil {
					panic(err2)
				}
			}()
		*/
	} else if err != nil {
		return
	}

	//db, err = sql.Open("sqlite", file+"?_busy_timeout=5000")
	db, err = zqlite.OpenSynchronizedClosingDB(file, "", BusyTimeout)
	if err != nil {
		return
	}

	// err = db.Open()
	// if err != nil {
	// 	return
	// }
	//logger.Debug("opened db", "file", file)
	//defer db.Close()

	/*
		db.SetMaxOpenConns(5)

		// Config to increase DB speed : temp objets and transaction journal stored in memory.
		_, err = db.Exec(`
			PRAGMA TEMP_STORE = MEMORY;
			PRAGMA JOURNAL_MODE = MEMORY;
			PRAGMA SYNCHRONOUS = OFF;
			PRAGMA LOCKING_MODE = NORMAL;
		`)
	*/
	return
}

func IsInitialized(db *zql.SynchronizedDB) (bool, error) {
	row := db.QueryRow(`
		SELECT count(name), coalesce(group_concat(coalesce(name, 'NIL')), 'NULL')
		FROM sqlite_schema
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name IS NOT NULL;
	`)
	var count int
	var names string
	err := row.Scan(&count, &names)
	if err != nil {
		return false, err
	}

	logger.Debug("cmdt sqlite tables", "count", count, "file", db.FileLockPath(), "tables", names)
	return count > 0, nil
}

/*
	func dbInit(db *zql.SynchronizedDB) (err error) {
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

			CREATE TABLE IF NOT EXISTS config (
				suite TEXT UNIQUE NOT NULL,
				serialized BLOB NOT NULL
			);

		`)

		logger.Warn("initialized db")
		return
	}
*/

func IsBusyError(err error) bool {
	return strings.Contains(err.Error(), "SQLITE_BUSY") || strings.Contains(err.Error(), "cannot start a transaction within a transaction")
}
