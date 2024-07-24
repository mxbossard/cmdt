package utils

import (
	"bytes"
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/errorz"
	"mby.fr/utils/trust"
	"mby.fr/utils/zlog"
)

var logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

func InitSeq(pathes ...string) (err error) {
	seqFilepath := filepath.Join(pathes...)
	err = os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		err = fmt.Errorf("cannot initialize seq file (%s): %w", seqFilepath, err)
	}
	return
}

func IncrementSeq(pathes ...string) (seq uint32) {
	// return an increment for test indexing
	seqFilepath := filepath.Join(pathes...)

	file, err := os.OpenFile(seqFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		errorz.Fatalf("cannot open seq file (%s) to increment: %s", seqFilepath, err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil && err != io.EOF {
		errorz.Fatalf("cannot read seq file (%s) to increment: %s", seqFilepath, err)
	}
	if strSeq == "" {
		seq = 0
	} else {
		var i int
		i, err = strconv.Atoi(strSeq)
		if err != nil {
			errorz.Fatalf("cannot convert seq file (%s) to an integer to increment: %s", seqFilepath, err)
		}
		seq = uint32(i)
	}

	newSec := seq + 1
	_, err = file.WriteAt([]byte(fmt.Sprint(newSec)), 0)
	if err != nil {
		errorz.Fatalf("cannot write seq file (%s) to increment: %s", seqFilepath, err)
	}

	//fmt.Printf("Incremented seq(%s %s %s): %d => %d\n", testSuite, token, filename, seq, newSec)
	return newSec
}

func ReadSeq(pathes ...string) (c uint32) {
	// return the count of run test
	seqFilepath := filepath.Join(pathes...)

	file, err := os.OpenFile(seqFilepath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		errorz.Fatalf("cannot open seq file (%s) to read: %s", seqFilepath, err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil {
		if err == io.EOF {
			return 0
		}
		errorz.Fatalf("cannot read seq file (%s) to read: %s", seqFilepath, err)
	}
	var i int
	i, err = strconv.Atoi(strSeq)
	if err != nil {
		errorz.Fatalf("cannot convert seq file (%s) as an integer to read: %s", seqFilepath, err)
	}
	c = uint32(i)
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
	logger.Trace("Found a token in env: " + token)
	return
}

func ReadEnvPpid() (ppid int) {
	// Search uniqKey in env
	if ok, ppidEnv := ReadEnvValue(model.ContextPpidEnvVarName); ok {
		var err error
		ppid, err = strconv.Atoi(ppidEnv)
		if err != nil {
			panic(err)
		}
		logger.Trace("found env ppid", "ppid", ppid)
	} else {
		ppid = os.Getppid()
		logger.Trace("env ppid not found", "ppid", ppid)
	}
	return
}

func ReadEnvTestSeq() (seq uint16) {
	// Search uniqKey in env
	if ok, seqEnv := ReadEnvValue(model.ContextTestSeqEnvVarName); ok {
		s, err := strconv.Atoi(seqEnv)
		if err != nil {
			panic(err)
		}
		seq = uint16(s)
		logger.Trace("found env seq", "seq", seq)
	} else {
		logger.Trace("env ppid not found", "seq", seq)
	}
	return
}

func ForgeContextualToken(token string) (string, error) {
	if token == "" {
		token = ReadEnvToken()
	}
	if token == "" {
		// If no token supplied use Workspace dir + ppid to forge tmp directory path
		workDirPath, err := os.Getwd()
		if err != nil {
			//errorz.Fatalf("cannot find workspace dir: %s", err)
			return "", fmt.Errorf("cannot find workspace dir: %w", err)
		}

		ppid := ReadEnvPpid()
		ppidStr := fmt.Sprintf("%d", ppid)
		ppidStartTime, err := GetProcessStartTime(ppid)
		if err != nil {
			//errorz.Fatalf("cannot find parent process start time: %s", err)
			return "", fmt.Errorf("cannot find parent process start time: %w", err)
		}
		ppidStartTimeStr := fmt.Sprintf("%d", ppidStartTime)
		token, err = trust.SignStrings(workDirPath, "--", ppidStr, "--", ppidStartTimeStr)
		if err != nil {
			err = fmt.Errorf("cannot hash workspace dir: %w", err)
			return "", err
		}
		logger.Trace("token signature", "workDirPath", workDirPath, "token", token, "ppidStr", ppidStr)
		//log.Printf("contextual token: %s base on workDirPath: %s and ppid: %s\n", token, workDirPath, ppid)
	}

	return token, nil
}

func IsolatedToken(token, isolation string) string {
	if isolation != "" {
		return fmt.Sprintf("%s_%s", token, isolation)
	}
	return token
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

func WhichCmd(cmd string) (absolute string, err error) {
	exec := cmdz.Sh("which", cmd).CombinedOutputs().AddEnviron(os.Environ()...)
	rc, err := exec.BlockRun()
	if err != nil {
		err = fmt.Errorf("cannot which command %s: %w", cmd, err)
		return
	} else if rc > 0 {
		err = fmt.Errorf("cannot which command %s: %s", cmd, exec.StdoutRecord())
		return
	}
	absolute = strings.TrimSpace(exec.StdoutRecord())
	return
}

func IsWithinContainer() (ok bool) {
	ok, _ = ReadEnvValue(model.EnvContainerIdKey)
	return
}
