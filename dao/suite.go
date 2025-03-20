package dao

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"time"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/zql"
)

func NewSuite(db *zql.SynchronizedDB, init bool) (d Suite, err error) {
	d.db = db
	if init {
		err = d.init()
	}
	return
}

func outcomeOrder(o model.Outcome) int {
	switch o {
	case model.PASSED:
		return 10
	case model.FAILED:
		return 20
	case model.TIMEOUT:
		return 30
	case model.ERRORED:
		return 40
	case model.IGNORED:
		return 50
	default:
		panic("bad outcome")
	}
	return 99
}

type Suite struct {
	db *zql.SynchronizedDB
}

func (d Suite) init() (err error) {
	res, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS suite (
			name TEXT UNIQUE NOT NULL,
			config BLOB NOT NULL,
			startTime INTEGER NOT NULL DEFAULT 0,
			seq INTEGER NOT NULL DEFAULT 0,
			tooMuch INTEGER NOT NULL DEFAULT 0,
			endTime INTEGER NOT NULL DEFAULT 0,
			outcome TEXT NOT NULL DEFAULT 'Z',
			outcomeOrder INTEGER DEFAULT 0,
			reported INTEGER NOT NULL DEFAULT 0,
			kept INTEGER NOT NULL DEFAULT 0
		);
	`)
	count, _ := res.RowsAffected()
	logger.Trace("init suite DAO", "rows affected", count)
	return
}

func (d Suite) NextSeq(suite string) (seq uint16, err error) {
	p := logger.PerfTimer("suite", suite, "filelock", d.db.FileLockPath())
	defer p.End()

	err = d.db.Lock()
	if err != nil {
		return
	}
	defer d.db.Unlock()

	//time.Sleep(time.Second)

	row := d.db.QueryRow(`
		SELECT s.seq
		FROM suite s
		WHERE s.name = ?
	`, suite)
	err = row.Scan(&seq)
	if err != nil {
		return
	}

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	logger.Debug("nextSeq()", "suite", suite, "before", seq)
	seq++
	_, err = tx.Exec(`
		UPDATE suite SET seq = ? 
		WHERE name = ?
	`, seq, suite)
	if err != nil {
		return
	}
	err = tx.Commit()
	if err != nil {
		return
	}
	logger.Debug("nextSeq()", "suite", suite, "after", seq)

	row = d.db.QueryRow(`
		SELECT s.seq
		FROM suite s
		WHERE s.name = ?
	`, suite)
	err = row.Scan(&seq)
	if err != nil {
		return
	}
	logger.Debug("nextSeq()", "suite", suite, "reSelect", seq)

	return
}

func (d Suite) IncrementTooMuchCount(suite string) (seq uint16, err error) {
	p := logger.PerfTimer("suite", suite, "filelock", d.db.FileLockPath())
	defer p.End()

	err = d.db.Lock()
	if err != nil {
		return
	}
	defer d.db.Unlock()

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
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

func (d Suite) NotReportedTestCount() (n uint16, err error) {
	p := logger.PerfTimer()
	defer p.End()

	row := d.db.QueryRow(`
		SELECT coalesce(sum(s.seq), 0)
		FROM suite s
		WHERE s.outcome <> 'Z'
	`)
	err = row.Scan(&n)
	return
}

func (d Suite) TestCount(suite string) (n uint16, err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End("n", n)

	row := d.db.QueryRow(`
		SELECT coalesce(max(s.seq), 0)
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&n)
	return
}

func (d Suite) TooMuchCount(suite string) (n uint16, err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	row := d.db.QueryRow(`
		SELECT s.tooMuch
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&n)
	return
}

func (d Suite) UpdateSuiteStartTime(suite string, start time.Time) (err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	micros := start.UnixMicro()
	_, err = d.db.Exec(`
		UPDATE suite SET startTime = ?
		WHERE name = ?
	`, micros, suite)
	return
}

func (d Suite) UpdateSuiteEndTime(suite string, end time.Time) (err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	micros := end.UnixMicro()
	_, err = d.db.Exec(`
			UPDATE suite SET endTime = ?
			WHERE name = ?
		`, micros, suite)
	return
}

func (d Suite) UpdateSuiteOutcome(suite string, outcome model.Outcome) (err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	order := outcomeOrder(outcome)
	_, err = d.db.Exec(`
		UPDATE suite SET outcome = ?, outcomeOrder = ?
		WHERE name = ? AND outcome > ?
	`, outcome, order, suite, outcome)
	return
}

func (d Suite) MarkSuiteReported(suite string, kept bool) (err error) {
	p := logger.PerfTimer("suite", suite, "kept", kept)
	defer p.End()

	_, err = d.db.Exec(`
		UPDATE suite SET reported = 1, kept = ?
		WHERE name = ?
	`, kept, suite)
	return
}

func (d Suite) IsSuiteReported(suite string) (exists, reported, kept bool, err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	row := d.db.QueryRow(`
		SELECT s.reported, s.kept
		FROM suite s
		WHERE s.name = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&reported, &kept)
	if err == sql.ErrNoRows {
		// Suite do not exists
		return false, false, false, nil
	} else {
		exists = true
	}
	return
}

func (d Suite) DeleteSuite(suite string) (err error) {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	_, err = d.db.Exec(`
		DELETE FROM suite
		WHERE name = ?
	`, suite)
	return
}

func (d Suite) ListPassedFailedErrored() (suites []string, err error) {
	p := logger.PerfTimer()
	defer p.End()

	rows, err := d.db.Query(`
		SELECT s.name
		FROM suite s
		WHERE s.startTime IS NOT NULL
		ORDER BY s.outcomeOrder ASC, s.startTime ASC
	`) // s.outcome IN ('PASSED', 'FAILED', 'ERRORED') AND
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
	p := logger.PerfTimer()
	defer p.End()

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
	p := logger.PerfTimer("testSuite", testSuite)
	defer p.End()

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
	return
}

func (d Suite) SaveGlobalConfig(cfg model.Config) (err error) {
	p := logger.PerfTimer()
	defer p.End()

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
	p := logger.PerfTimer("testSuite", testSuite)
	defer p.End()

	serializedConfig, err := serializeConfig(cfg)
	if err != nil {
		return
	}

	err = d.db.Lock()
	if err != nil {
		return
	}
	defer d.db.Unlock()

	//time.Sleep(1 * time.Second)
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
