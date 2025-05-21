package dao

import (
	"database/sql"
	"errors"

	"github.com/mxbossard/utilz/zqlite"
)

func NewGlobal(db *zqlite.SynchronizedDB, init bool) (d Global, err error) {
	d.db = db
	if init {
		err = d.init()
	}
	return
}

type Global struct {
	db *zqlite.SynchronizedDB
}

func (d Global) init() (err error) {
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS global (
			daemonPid INTEGER NULL
		);
	`)
	return
}

func (d Global) SaveDaemonPid(pid int) (err error) {
	p := logger.PerfTimer("pid", pid)
	defer p.End()

	_, err = d.db.Exec("INSERT OR REPLACE INTO global(daemonPid) VALUES (@pid);",
		sql.Named("pid", pid))

	return
}

func (d Global) ClearDaemonPid(pid int) (err error) {
	p := logger.PerfTimer("pid", pid)
	defer p.End()

	_, err = d.db.Exec("UPDATE global set daemonPid = 0 WHERE daemonPid = @pid;",
		sql.Named("pid", pid))

	return
}

func (d Global) GetDaemonPid() (pid int, err error) {
	p := logger.PerfTimer()
	defer p.End()

	row := d.db.QueryRow(`
		SELECT daemonPid
		FROM global
	`)
	err = row.Scan(&pid)
	if errors.Is(err, sql.ErrNoRows) {
		// No row found => Return pid = 0
		err = nil
	}

	if err != nil {
		return
	}

	return
}
