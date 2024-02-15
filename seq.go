package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func InitSeq(testSuite, token string) {
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	seqFilepath := filepath.Join(tmpDir, TestSequenceFilename)
	err := os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		log.Fatalf("cannot initialize seq file (%s): %s", seqFilepath, err)
	}
	seqFilepath = filepath.Join(tmpDir, IgnoredSequenceFilename)
	err = os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		log.Fatalf("cannot initialize seq file (%s): %s", seqFilepath, err)
	}
}

func IncrementSeq(testSuite, token, filename string) (seq int) {
	// return an increment for test indexing
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	seqFilepath := filepath.Join(tmpDir, filename)

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

func ReadSeq(testSuite, token, filename string) (c int) {
	// return the count of run test
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	seqFilepath := filepath.Join(tmpDir, filename)

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
