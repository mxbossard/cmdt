package utils

import (
	"bytes"
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/trust"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

/*
func Fatal(testSuite, token string, v ...any) {
	tmpDir, err := TestsuiteDirectoryPath(testSuite, token)
	if err != nil {
		log.Fatal(err)
	}
	IncrementSeq(tmpDir, model.ErrorSequenceFilename)
	log.Fatal(v...)
}

func Fatalf(testSuite, token, format string, v ...any) {
	log.Fatal(testSuite, token, fmt.Sprintf(format, v...))
}
*/

/*
func SuiteError(testSuite, token string, v ...any) error {
	return SuiteErrorf(testSuite, token, "%s", fmt.Sprint(v...))
}

func SuiteErrorf(testSuite, token, format string, v ...any) error {
	IncrementSeq(testSuite, token, model.ErrorSequenceFilename)
	return fmt.Errorf(format, v...)
}
*/

func InitSeq(pathes ...string) (err error) {
	seqFilepath := filepath.Join(pathes...)
	err = os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		err = fmt.Errorf("cannot initialize seq file (%s): %w", seqFilepath, err)
	}
	return
}

func IncrementSeq(pathes ...string) (seq int) {
	// return an increment for test indexing
	//tmpDir := testsuiteDirectoryPath(testSuite, token)
	seqFilepath := filepath.Join(pathes...)

	file, err := os.OpenFile(seqFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("cannot open seq file (%s): %s", seqFilepath, err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil && err != io.EOF {
		log.Fatalf("cannot read seq file (%s): %s", seqFilepath, err)
	}
	if strSeq == "" {
		seq = 0
	} else {
		seq, err = strconv.Atoi(strSeq)
		if err != nil {
			log.Fatalf("cannot read seq file (%s) as an integer: %s", seqFilepath, err)
		}
	}

	newSec := seq + 1
	_, err = file.WriteAt([]byte(fmt.Sprint(newSec)), 0)
	if err != nil {
		log.Fatalf("cannot write seq file (%s): %s", seqFilepath, err)
	}

	//fmt.Printf("Incremented seq(%s %s %s): %d => %d\n", testSuite, token, filename, seq, newSec)
	return newSec
}

func ReadSeq(pathes ...string) (c int) {
	// return the count of run test
	//tmpDir := testsuiteDirectoryPath(testSuite, token)
	seqFilepath := filepath.Join(pathes...)

	file, err := os.OpenFile(seqFilepath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		log.Fatalf("cannot open seq file (%s): %s", seqFilepath, err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil {
		if err == io.EOF {
			return 0
		}
		log.Fatalf("cannot read seq file (%s): %s", seqFilepath, err)
	}
	c, err = strconv.Atoi(strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file (%s) as an integer: %s", seqFilepath, err)
	}
	return
}

func ForgeUuid() (uuid string, err error) {
	b := make([]byte, 16)
	_, err = cryptorand.Read(b)
	if err != nil {
		return
	}
	uuid = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}

func GetProcessStartTime(pid int) (int64, error) {
	// Index of the starttime field. See proc(5).
	const StartTimeIndex = 21

	fname := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return 0, err
	}

	fields := bytes.Fields(data)
	if len(fields) < StartTimeIndex+1 {
		return 0, fmt.Errorf("invalid /proc/[pid]/stat format: too few fields: %d", len(fields))
	}

	s := string(fields[StartTimeIndex])
	starttime, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid starttime: %w", err)
	}

	return starttime, nil
}

func ReadEnvValue(key string) (ok bool, value string) {
	for _, env := range os.Environ() {

		if ok = strings.HasPrefix(env, key+"="); ok {
			splitted := strings.Split(env, "=")
			value = strings.Join(splitted[1:], "")
			return
		}
	}
	return
}

func ReadEnvToken() (token string) {
	// Search uniqKey in env
	_, token = ReadEnvValue(model.ContextTokenEnvVarName)
	logger.Debug("Found a token in env: " + token)
	return
}

func ForgeContextualToken() (string, error) {
	token := ReadEnvToken()
	if token != "" {
		return token, nil
	}
	// If no token supplied use Workspace dir + ppid to forge tmp directory path
	workDirPath, err := os.Getwd()
	if err != nil {
		//log.Fatalf("cannot find workspace dir: %s", err)
		return "", fmt.Errorf("cannot find workspace dir: %w", err)
	}
	ppid := os.Getppid()
	ppidStr := fmt.Sprintf("%d", ppid)
	ppidStartTime, err := GetProcessStartTime(ppid)
	if err != nil {
		//log.Fatalf("cannot find parent process start time: %s", err)
		return "", fmt.Errorf("cannot find parent process start time: %w", err)
	}
	ppidStartTimeStr := fmt.Sprintf("%d", ppidStartTime)
	token, err = trust.SignStrings(workDirPath, "--", ppidStr, "--", ppidStartTimeStr)
	if err != nil {
		err = fmt.Errorf("cannot hash workspace dir: %w", err)
	}
	//log.Printf("contextual token: %s base on workDirPath: %s and ppid: %s\n", token, workDirPath, ppid)
	return token, err
}

func IsShellBuiltin(cmd string) (ok bool, err error) {
	exec := cmdz.Sh("type", cmd).CombinedOutputs().AddEnviron(os.Environ()...)
	rc, err := exec.BlockRun()
	if err != nil {
		err = fmt.Errorf("cannot evaluate if command %s is a shell builtin: %w", cmd, err)
		return
	} else if rc > 0 {
		if strings.Contains(exec.StdoutRecord(), "not found") {
			err = fmt.Errorf("command %s not found in path: %s", cmd, exec.StdoutRecord())
			return
		} else {
			err = fmt.Errorf("cannot evaluate if command %s is a shell builtin: %s", cmd, exec.StdoutRecord())
			return
		}
	}
	ok = strings.Contains(exec.StdoutRecord(), "shell builtin")
	return
}

func IsWithinContainer() (ok bool) {
	ok, _ = ReadEnvValue(model.EnvContainerIdKey)
	return
}
