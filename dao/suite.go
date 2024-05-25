package dao

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"time"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/zql"
)

func NewSuite(db *zql.SynchronizedDB) (d Suite, err error) {
	d.db = db
	err = d.init()
	return
}

type Suite struct {
	db *zql.SynchronizedDB
}

func (d Suite) init() (err error) {
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS suite (
			name TEXT UNIQUE NOT NULL,
			config BLOB NOT NULL,
			startTime INTEGER NOT NULL DEFAULT 0,
			seq INTEGER NOT NULL DEFAULT 0,
			tooMuch INTEGER NOT NULL DEFAULT 0,
			endTime INTEGER NOT NULL DEFAULT 0,
			outcome TEXT NOT NULL DEFAULT ''
		);
	`)
	return
}

func (d Suite) NextSeq(suite string) (seq uint16, err error) {
	logger.Debug("NextSeq() start", "suite", suite, "filelock", d.db.FileLockPath())
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	row := tx.QueryRow(`
		SELECT s.seq
		FROM suite s
		WHERE s.name = ?
	`, suite)
	err = row.Scan(&seq)
	if err != nil {
		return
	}
	seq++
	_, err = tx.Exec(`
		UPDATE suite SET seq = ? 
		WHERE name = ?
	`, seq, suite)
	err = tx.Commit()
	if err != nil {
		return
	}
	logger.Debug("NextSeq() end", "suite", suite, "seq", seq, "filelock", d.db.FileLockPath())

	return
}

func (d Suite) IncrementTooMuchCount(suite string) (seq uint16, err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	row := tx.QueryRow(`
		SELECT tooMuch
		FROM suite s
		WHERE s.name = ?
	`, suite)
	err = row.Scan(&seq)
	if err != nil {
		return
	}
	_, err = tx.Exec(`
		UPDATE suite SET tooMuch = ? 
		WHERE name = ?
	`, seq+1, suite)
	err = tx.Commit()
	return
}

func (d Suite) TestCount(suite string) (n uint16, err error) {
	logger.Debug("TestCount() start", "suite", suite)
	row := d.db.QueryRow(`
		SELECT coalesce(max(s.seq), 0)
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&n)
	logger.Debug("TestCount() end", "suite", suite)
	return
}

func (d Suite) TooMuchCount(suite string) (n uint16, err error) {
	logger.Debug("TooMuchCount() start", "suite", suite)
	row := d.db.QueryRow(`
		SELECT s.tooMuch
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&n)
	logger.Debug("TooMuchCount() end", "suite", suite)
	return
}

func (d Suite) UpdateSuiteStartTime(suite string, start time.Time) (err error) {
	logger.Debug("UpdateSuiteStartTime() start", "suite", suite)
	micros := start.UnixMicro()
	_, err = d.db.Exec(`
		UPDATE suite SET startTime = ?
		WHERE name = ?
	`, micros, suite)
	logger.Debug("UpdateSuiteStartTime() end", "suite", suite)
	return
}

func (d Suite) UpdateSuiteEndTime(suite string, end time.Time) (err error) {
	logger.Debug("UpdateSuiteEndTime() start", "suite", suite)
	/*
		q, err := d.db.Prepare(`
			UPDATE suite SET endTime = ?
			WHERE name = ?
		`)
		if err != nil {
			return
		}*/
	micros := end.UnixMicro()

	_, err = d.db.Exec(`
			UPDATE suite SET endTime = ?
			WHERE name = ?
		`, micros, suite)

	//_, err = q.Exec(micros, suite)

	logger.Debug("UpdateSuiteEndTime() end", "suite", suite)
	return
}

func (d Suite) UpdateSuiteOutcome(suite string, outcome model.Outcome) (err error) {
	logger.Debug("UpdateSuiteOutcome() start", "suite", suite)
	_, err = d.db.Exec(`
		UPDATE suite SET outcome = ?
		WHERE name = ?
	`, outcome, suite)
	logger.Debug("UpdateSuiteOutcome() end", "suite", suite)
	return
}

func (d Suite) DeleteSuite(suite string) (err error) {
	logger.Debug("DeleteSuite() start", "suite", suite)
	_, err = d.db.Exec(`
		DELETE FROM suite
		WHERE name = ?
	`, suite)
	logger.Debug("DeleteSuite() end", "suite", suite)
	return
}

func (d Suite) ListPassedFailedErrored() (suites []string, err error) {
	logger.Debug("ListPassedFailedErrored() start")
	rows, err := d.db.Query(`
		SELECT s.name
		FROM suite s
		WHERE s.startTime IS NOT NULL
		ORDER BY s.outcome DESC, s.startTime ASC
	`)
	// s.outcome IN ('PASSED', 'FAILED', 'ERRORED') AND
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var suiteName string
		err = rows.Scan(&suiteName)
		if err != nil {
			return
		}
		suites = append(suites, suiteName)
	}
	logger.Debug("ListPassedFailedErrored() end")
	return
}

func (d Suite) FindGlobalConfig() (cfg *model.Config, err error) {
	logger.Debug("FindGlobalConfig() start")
	var serializedConfig []byte
	row := d.db.QueryRow(`
		SELECT s.config 
		FROM suite s
		WHERE name = '';
	`)
	err = row.Scan(&serializedConfig)
	if err == sql.ErrNoRows {
		err = nil
		return
	} else if err != nil {
		return
	}
	cfg = &model.Config{}
	err = deserializeConfig(serializedConfig, cfg)
	logger.Debug("FindGlobalConfig() end")
	return
}

func (d Suite) FindSuiteConfig(testSuite string) (cfg *model.Config, err error) {
	logger.Debug("FindSuiteConfig() start", "testSuite", testSuite)
	var serializedConfig []byte
	var startTime, endTime, seq int64
	var outcome string
	row := d.db.QueryRow(`
		SELECT s.config, s.startTime, s.endTime, s.outcome, s.seq 
		FROM suite s
		WHERE name = @suite;
	`, sql.Named("suite", testSuite))
	err = row.Scan(&serializedConfig, &startTime, &endTime, &outcome, &seq)
	if err == sql.ErrNoRows {
		err = nil
		return
	} else if err != nil {
		return
	}
	cfg = &model.Config{}
	err = deserializeConfig(serializedConfig, cfg)
	logger.Debug("FindSuiteConfig() end", "testSuite", testSuite)
	return
}

func (d Suite) SaveGlobalConfig(cfg model.Config) (err error) {
	logger.Debug("SaveGlobalConfig() start")
	serializedConfig, err := serializeConfig(cfg)
	if err != nil {
		return
	}
	_, err = d.db.Exec("INSERT OR REPLACE INTO suite(name, config) VALUES ('', @serCfg);",
		sql.Named("serCfg", serializedConfig),
	)
	logger.Debug("SaveGlobalConfig() end")
	return
}

func (d Suite) SaveSuiteConfig(testSuite string, cfg model.Config) (err error) {
	logger.Debug("SaveSuiteConfig() start", "testSuite", testSuite)
	serializedConfig, err := serializeConfig(cfg)
	if err != nil {
		return
	}

	_, err = d.db.Exec(`INSERT OR IGNORE INTO suite(name, config) VALUES (@suite, '');`, sql.Named("suite", testSuite))
	if err != nil {
		return
	}

	if cfg.SuiteStartTime.IsPresent() {
		micros := cfg.SuiteStartTime.Get().UnixMicro()
		_, err = d.db.Exec(`
				UPDATE suite SET config = @serCfg, startTime = @startTime
				WHERE name = @suite;`,
			sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
			sql.Named("startTime", micros),
		)
	} else {
		_, err = d.db.Exec(`
				UPDATE suite SET config = @serCfg
				WHERE name = @suite;`,
			sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
		)
	}
	if err != nil {
		return
	}

	row := d.db.QueryRow(`
		SELECT s.seq
		FROM suite s
		WHERE s.name = ?
	`, testSuite)
	var seq uint16
	err = row.Scan(&seq)
	if err != nil {
		return
	}
	logger.Debug("SaveSuiteConfig() end", "suite", testSuite, "seq", seq)
	return
}

func serializeConfig(cfg model.Config) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(cfg)
	if err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}

func deserializeConfig(b []byte, cfg *model.Config) (err error) {
	buf := bytes.NewReader(b)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(cfg)
	return
}
