package dao

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/collections"
	"mby.fr/utils/zql"
)

const (
	CMD_AND_ARGS_SEPARATOR = ","
)

func NewTest(db *zql.SynchronizedDB) (d Test, err error) {
	d.db = db
	err = d.init()
	return
}

type Test struct {
	db *zql.SynchronizedDB
}

func (d Test) init() (err error) {
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS tested (
			suite TEXT NOT NULL,
			seq INTEGER NOT NULL,
			name TEXT NOT NULL,
			cmdAndArgs TEXT NOT NULL,
			outcome TEXT NOT NULL,
			errorMsg TEXT NOT NULL DEFAULT '',
			duration INTEGER NOT NULL DEFAULT -1,
			passed INTEGER NOT NULL DEFAULT 0,
			failed INTEGER NOT NULL DEFAULT 0,
			errored INTEGER NOT NULL DEFAULT 0,
			ignored INTEGER NOT NULL DEFAULT 0,
			exitCode INTEGER NOT NULL DEFAULT -1,
			stdout TEXT NOT NULL DEFAULT '',
			stderr TEXT NOT NULL DEFAULT '',
			report TEXT NOT NULL DEFAULT '',

			PRIMARY KEY (suite, seq),
			FOREIGN KEY(suite) REFERENCES suite(name)
		);

		CREATE INDEX IF NOT EXISTS tested_suite_idx ON tested(suite);

		CREATE TABLE IF NOT EXISTS assertion_result (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suite TEXT NOT NULL,
			seq INTEGER NOT NULL,
			prefix TEXT NOT NULL,
			name TEXT NOT NULL,
			op TEXT NOT NULL DEFAULT '',
			expected TEXT NOT NULL DEFAULT '',
			value TEXT NOT NULL DEFAULT '',
			errorMsg TEXT NOT NULL DEFAULT '',
			success INTEGER NOT NULL,

			FOREIGN KEY(suite, seq) REFERENCES tested(suite, seq)
		);	

		CREATE INDEX IF NOT EXISTS assertion_result_fk_idx ON assertion_result(suite, seq);
	`)
	return
}

func (d Test) GetSuiteOutcome(suite string) (outcome model.SuiteOutcome, err error) {
	var passedCount, failedCount, erroredCount, ignoredCount uint32
	var testName, cmdAndArgs, testOc, stdout, stderr, testErrorMsg, prefix, assertName, op, expected, value, assertErrorMsg string
	var success bool
	var startTime, endTime, testDuration int64
	var seq uint16
	var exitCode int16
	//var failedAssertionsMessages []string

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		SELECT sum(passed), sum(failed), sum(errored), sum(ignored)
		FROM tested
		WHERE suite = @suite
	`, sql.Named("suite", suite))
	err = row.Scan(&passedCount, &failedCount, &erroredCount, &ignoredCount)
	if err != nil {
		return
	}

	rows, err := tx.Query(`
		SELECT t.seq, t.name, t.cmdAndArgs, t.outcome, t.exitCode, t.errorMsg, t.duration, 
			t.stdout, t.stderr,	COALESCE(a.prefix, ''), COALESCE(a.name, ''), COALESCE(a.op, ''), 
			COALESCE(a.expected, ''), COALESCE(a.value, ''), COALESCE(a.errorMsg, ''), 
			COALESCE(a.success, 0)
		FROM tested t LEFT JOIN assertion_result a ON a.suite = t.suite AND a.seq = t.seq
		WHERE t.suite = @suite 
		ORDER BY t.seq ASC
	`, sql.Named("suite", suite))
	if err != nil {
		return
	}
	defer rows.Close()

	testOutcomeBySeq := make(map[uint16]model.TestOutcome)
	for rows.Next() {
		if err = rows.Scan(&seq, &testName, &cmdAndArgs, &testOc, &exitCode, &testErrorMsg,
			&testDuration, &stdout, &stderr, &prefix, &assertName, &op, &expected, &value,
			&assertErrorMsg, &success); err != nil {
			return
		}
		cmdAndArgsArray := strings.Split(cmdAndArgs, CMD_AND_ARGS_SEPARATOR)
		var testOutcome model.TestOutcome
		var ok bool
		if testOutcome, ok = testOutcomeBySeq[seq]; !ok {
			sign := model.TestSignature{
				TestSuite:  suite,
				Seq:        seq,
				TestName:   testName,
				CmdAndArgs: cmdAndArgsArray,
			}
			testOutcome = model.TestOutcome{
				TestSignature: sign,
				Outcome:       model.Outcome(testOc),
				ExitCode:      exitCode,
				Err:           fmt.Errorf(testErrorMsg),
				Duration:      time.Duration(testDuration * 1000),
				Stdout:        stdout,
				Stderr:        stderr,
			}
			testOutcomeBySeq[seq] = testOutcome
		}

		if assertName != "" {
			res := model.NewAssertionResult(prefix, assertName, op, expected, value, success, assertErrorMsg)
			testOutcome.AssertionResults = append(testOutcome.AssertionResults, res)
			testOutcomeBySeq[seq] = testOutcome
		}
	}

	row = tx.QueryRow(`
		SELECT startTime, endTime
		FROM suite
		WHERE name = @suite 
	`, sql.Named("suite", suite))
	err = row.Scan(&startTime, &endTime)
	if err != nil {
		return
	}

	err = tx.Commit()
	if err != nil {
		return
	}
	testCount := passedCount + failedCount + erroredCount + ignoredCount
	duration := time.Duration((endTime - startTime) * 1000)
	var ocm model.Outcome
	if testCount == passedCount {
		ocm = model.PASSED
	} else if testCount == ignoredCount {
		ocm = model.IGNORED
	} else if erroredCount > 0 {
		ocm = model.ERRORED
	} else {
		ocm = model.FAILED
	}

	outcome.TestSuite = suite
	outcome.Duration = duration
	outcome.TestCount = testCount
	outcome.PassedCount = passedCount
	outcome.FailedCount = failedCount
	outcome.ErroredCount = erroredCount
	outcome.IgnoredCount = ignoredCount
	outcome.Outcome = ocm
	//outcome.FailureReports = failedAssertionsMessages
	outcome.TestOutcomes = collections.MapOrderedValues(testOutcomeBySeq)

	return
}

func (d Test) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	seq := outcome.Seq
	suite := outcome.TestSuite
	name := outcome.TestName
	cmdAndArgs := strings.Join(outcome.CmdAndArgs, CMD_AND_ARGS_SEPARATOR)
	errorMsg := ""
	if outcome.Err != nil {
		errorMsg = outcome.Err.Error()
	}
	micros := outcome.Duration.Microseconds()
	oc := outcome.Outcome
	passed := outcome.Outcome == model.PASSED
	ignored := outcome.Outcome == model.IGNORED
	failed := outcome.Outcome == model.FAILED
	errored := outcome.Outcome == model.ERRORED
	exitCode := outcome.ExitCode
	stdout := outcome.Stdout
	stderr := outcome.Stderr
	report := "NO REPORT"

	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO tested(suite, seq, name, cmdAndArgs, errorMsg, duration, outcome, passed, ignored, failed, errored, exitCode, stdout, stderr, report) 
		VALUES (@suite, @seq, @name, @cmdAndArgs, @errorMsg, @micros, @outcome, @passed, @ignored, @failed, @errored, @exitCode, @stdout, @stderr, @report);
	`, sql.Named("suite", suite), sql.Named("seq", seq), sql.Named("name", name),
		sql.Named("cmdAndArgs", cmdAndArgs), sql.Named("errorMsg", errorMsg),
		sql.Named("micros", micros), sql.Named("outcome", oc),
		sql.Named("passed", passed), sql.Named("ignored", ignored),
		sql.Named("failed", failed), sql.Named("errored", errored),
		sql.Named("exitCode", exitCode), sql.Named("stdout", stdout),
		sql.Named("stderr", stderr), sql.Named("report", report))
	if err != nil {
		return
	}

	for _, res := range outcome.AssertionResults {
		rule := res.Rule
		rulePrefix := rule.Prefix
		ruleName := rule.Name
		ruleOp := rule.Op
		ruleExpected := rule.Expected
		value := res.Value
		errorMsg := res.ErrMessage
		success := res.Success
		_, err = tx.Exec(`
			INSERT INTO assertion_result(suite, seq, prefix, name, op, expected, value, errorMsg, success) 
			VALUES (@suite, @seq, @rulePrefix, @ruleName, @ruleOp, @ruleExpected, @value, @errorMsg, @success)
		`, sql.Named("suite", suite), sql.Named("seq", seq),
			sql.Named("rulePrefix", rulePrefix), sql.Named("ruleName", ruleName),
			sql.Named("ruleOp", ruleOp), sql.Named("ruleExpected", ruleExpected),
			sql.Named("value", value), sql.Named("errorMsg", errorMsg),
			sql.Named("success", success))
		if err != nil {
			return
		}
		log.Printf("Inserted assertion result (%s, %d), %v\n", suite, seq, res)
	}

	err = tx.Commit()

	return
}

func (d Test) ClearSuite(suite string) (err error) {
	_, err = d.db.Exec(`
		DELETE FROM assertion_result
		WHERE suite = @suite;
		DELETE FROM tested
		WHERE suite = @suite;
	`, sql.Named("suite", suite))
	return
}

/*
func (d Test) SaveTested(suite string, seq int, cfg model.Config) (err error) {
	// TODO
	return
}
*/
