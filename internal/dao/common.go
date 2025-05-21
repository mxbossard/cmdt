package dao

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	//_ "github.com/mattn/go-sqlite3"

	"github.com/mxbossard/utilz/zlog"
	"github.com/mxbossard/utilz/zqlite"
)

const (
	DbFileName  = "cmdt.sqlite"
	BusyTimeout = 10 * time.Second
)

var (
	logger = zlog.New()
)

func DbOpen(dirpath string) (db *zqlite.SynchronizedDB, err error) {
	file := filepath.Join(dirpath, DbFileName)

	_, err = os.Stat(file)
	if os.IsNotExist(err) {

	} else if err != nil {
		return
	}

	db, err = zqlite.OpenSynchronizedDB(file, "", BusyTimeout)
	if err != nil {
		return
	}

	return
}

func IsInitialized(db *zqlite.SynchronizedDB) (bool, error) {
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

func IsBusyError(err error) bool {
	return strings.Contains(err.Error(), "SQLITE_BUSY") || strings.Contains(err.Error(), "cannot start a transaction within a transaction")
}
