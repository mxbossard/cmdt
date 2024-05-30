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
	logger = zlog.NewColored() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
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
	db, err = zqlite.OpenSynchronizedDB(file, "", BusyTimeout)
	if err != nil {
		return
	}

	db.SetMaxOpenConns(1)

	// Config to increase DB speed : temp objets and transaction journal stored in memory.
	db.Exec(`
		PRAGMA TEMP_STORE = MEMORY;
		PRAGMA JOURNAL_MODE = MEMORY;
		PRAGMA SYNCHRONOUS = OFF;
		PRAGMA LOCKING_MODE = NORMAL;
	`)

	logger.Debug("opened db", "file", file)
	return
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
