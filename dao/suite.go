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
	d.init()
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
			startTime INTEGER NULL,
			seq INTEGER NOT NULL,
			endTime INTEGER NULL,
			outcome TEXT NULL
		);
	`)
	return
}

func (d Test) NextSeq(suite string) (seq int, err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	row := tx.QueryRow(`
		SELECT seq
		FROM suite
		WHERE name = ?
	`, suite)
	row.Scan(&seq)
	_, err = tx.Exec(`
		UPDATE suite SET seq = ? 
		WHERE name = ?
	`, seq+1, suite)
	err = tx.Commit()
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
	_, err = d.db.Exec(`
		SELECT name
		FROM suite
		WHERE outcome IN ('PASSED', 'FAILED', 'ERRORED') AND startTime IS NOT NULL
		ORDER BY outcome DESC, startTime ASC
	`)
	return
}

func (d Suite) FindGlobalConfig() (cfg *model.Config, err error) {
	logger.Warn("LoadGlobalConfig()")
	var serializedConfig []byte
	row := d.db.QueryRow(`
		SELECT config 
		FROM suite
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
	logger.Warn("LoadSuiteConfig()")
	var serializedConfig []byte
	row := d.db.QueryRow(`
		SELECT config 
		FROM suite
		WHERE name = @suite;
	`, sql.Named("suite", testSuite))
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

func (d Suite) SaveGlobalConfig(cfg model.Config) (err error) {
	logger.Warn("PersistGlobalConfig()")
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
	logger.Warn("PersistGlobalConfig()")
	serializedConfig, err := serializeConfig(cfg)
	if err != nil {
		return
	}
	if cfg.SuiteStartTime.IsPresent() {
		micros := cfg.SuiteStartTime.Get().UnixMicro()
		_, err = d.db.Exec("INSERT OR REPLACE INTO suite(name, config, startTime) VALUES (@suite, @serCfg, @startTime);",
			sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
			sql.Named("startTime", micros),
		)
	} else {
		_, err = d.db.Exec("INSERT OR REPLACE INTO suite(name, config) VALUES (@suite, @serCfg);",
			sql.Named("suite", testSuite), sql.Named("serCfg", serializedConfig),
		)
	}
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
