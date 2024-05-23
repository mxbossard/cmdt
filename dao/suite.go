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
	logger.Warn("incremented seq", "suite", suite, "seq", seq, "filelock", d.db.FileLockPath())

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
	row := d.db.QueryRow(`
		SELECT coalesce(max(s.seq), 0)
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&n)
	return
}

func (d Suite) TooMuchCount(suite string) (n uint16, err error) {
	row := d.db.QueryRow(`
		SELECT s.tooMuch
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&n)
	return
}

func (d Suite) UpdateStartTime(suite string, start time.Time) (err error) {
	micros := start.UnixMicro()
	_, err = d.db.Exec(`
		UPDATE suite SET startTime = ?
		WHERE name = ?
	`, micros, suite)
	return
}

func (d Suite) UpdateEndTime(suite string, end time.Time) (err error) {
	micros := end.UnixMicro()
	_, err = d.db.Exec(`
		UPDATE suite SET endTime = ?
		WHERE name = ?
	`, micros, suite)
	return
}

func (d Suite) UpdateOutcome(suite string, outcome model.Outcome) (err error) {
	_, err = d.db.Exec(`
		UPDATE suite SET outcome = ?
		WHERE name = ?
	`, outcome, suite)
	return
}

func (d Suite) Delete(suite string) (err error) {
	_, err = d.db.Exec(`
		DELETE FROM suite
		WHERE name = ?
	`, suite)
	return
}

func (d Suite) ListPassedFailedErrored() (suites []string, err error) {
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

	return
}

func (d Suite) FindGlobalConfig() (cfg *model.Config, err error) {
	logger.Debug("FindGlobalConfig()")
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
	return
}

func (d Suite) FindSuiteConfig(testSuite string) (cfg *model.Config, err error) {
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
	logger.Debug("FindSuiteConfig()", "cfg", *cfg)
	return
}

func (d Suite) SaveGlobalConfig(cfg model.Config) (err error) {
	logger.Warn("SaveGlobalConfig()")
	serializedConfig, err := serializeConfig(cfg)
	if err != nil {
		return
	}
	_, err = d.db.Exec("INSERT OR REPLACE INTO suite(name, config) VALUES ('', @serCfg);",
		sql.Named("serCfg", serializedConfig),
	)
	return
}

func (d Suite) SaveSuiteConfig(testSuite string, cfg model.Config) (err error) {
	logger.Debug("SaveSuiteConfig()", "cfg", cfg)
	serializedConfig, err := serializeConfig(cfg)
	if err != nil {
		return
	}

	_, err = d.db.Exec(`INSERT OR IGNORE INTO suite(name) VALUES (@suite);`, sql.Named("suite", testSuite))
	if err != nil {
		return
	}

	if cfg.SuiteStartTime.IsPresent() {
		micros := cfg.SuiteStartTime.Get().UnixMicro()

		_, err = d.db.Exec(`INSERT OR REPLACE INTO suite(seq, name, config, startTime) VALUES (
					(SELECT seq FROM suite WHERE name = @suite),
					@suite,
					@serCfg,
					@startTime
				);`,
			sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
			sql.Named("startTime", micros),
		)

		/*
			_, err = d.db.Exec(`
					UPDATE suite SET config = @serCfg, startTime = @startTime
					WHERE name = @suite;`,
				sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
				sql.Named("startTime", micros),
			)
		*/
	} else {

		_, err = d.db.Exec(`INSERT OR REPLACE INTO suite(seq, name, config) VALUES (
				(SELECT seq FROM suite WHERE name = @suite),
				@suite,
				@serCfg
				);`,
			sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
		)
		/*
			_, err = d.db.Exec(`
					UPDATE suite SET config = @serCfg
					WHERE name = @suite;`,
				sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
			)
		*/
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
	logger.Warn("saved suite config", "suite", testSuite, "seq", seq)
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
