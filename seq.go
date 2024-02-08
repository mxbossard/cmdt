package main

import (
	"fmt"
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
	if err != nil {
		log.Fatalf("cannot read seq file (%s) as an integer: %s", seqFilepath, err)
	}
	seq, err = strconv.Atoi(strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file (%s) as an integer: %s", seqFilepath, err)
	}

	newSec := seq + 1
	_, err = file.WriteAt([]byte(fmt.Sprint(newSec)), 0)
	if err != nil {
		log.Fatalf("cannot write seq file (%s): %s", seqFilepath, err)
	}

	//fmt.Printf("Incremented seq: %d => %d\n", seq, newSec)
	return newSec
}

func ReadSeq(testSuite, token, filename string) (c int) {
	// return the count of run test
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	seqFilepath := filepath.Join(tmpDir, filename)

	file, err := os.OpenFile(seqFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("cannot open seq file (%s): %s", seqFilepath, err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file (%s) as an integer: %s", seqFilepath, err)
	}
	c, err = strconv.Atoi(strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file (%s) as an integer: %s", seqFilepath, err)
	}
	return
}
