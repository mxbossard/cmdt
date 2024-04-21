package model

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

type OperationKind string

const (
	TestKind   = OperationKind("test")
	ReportKind = OperationKind("report")
)

type Operater interface {
	Kind() string
	Suite() string
	Seq() uint16
	Block() bool
	ExitCode() uint16
	SetExitCode(uint16)
	Id() uint16
	SetId(uint16)
	String() string
}

type OperationBase struct {
	//Token     string
	Type      string
	TestSuite string
	Sequence  uint16
	Blocking  bool
	exit      uint16
	id        uint16
}

func (o OperationBase) Kind() string {
	return o.Type
}

func (o OperationBase) Suite() string {
	return o.TestSuite
}

func (o OperationBase) Seq() uint16 {
	return o.Sequence
}

func (o OperationBase) Block() bool {
	return o.Blocking
}

func (o OperationBase) ExitCode() uint16 {
	return o.exit
}

func (o *OperationBase) SetExitCode(code uint16) {
	o.exit = code
}

func (o OperationBase) Id() uint16 {
	return o.id
}

func (o *OperationBase) SetId(id uint16) {
	o.id = id
}

func (o *OperationBase) String() string {
	return fmt.Sprintf("Op[type: %s ; id: %d]", o.Type, o.id)
}

type TestOp struct {
	OperationBase //`yaml:",inline"`
	Definition    TestDefinition
}

func TestOperation(suite string, seq uint16, blocking bool, def TestDefinition) TestOp {
	return TestOp{
		OperationBase: OperationBase{
			Type:      string(TestKind),
			TestSuite: suite,
			Sequence:  seq,
			Blocking:  blocking,
		},
		Definition: def,
	}
}

type ReportOp struct {
	OperationBase
	Definition ReportDefinition
}

func ReportOperation(suite string, blocking bool, def ReportDefinition) ReportOp {
	return ReportOp{
		OperationBase: OperationBase{
			Type:      string(ReportKind),
			TestSuite: suite,
			Blocking:  blocking,
		},
		Definition: def,
	}
}

type ReportAllOp struct {
	OperationBase
	Definition ReportDefinition
}

func ReportAllOperation(blocking bool, def ReportDefinition) ReportAllOp {
	return ReportAllOp{
		OperationBase: OperationBase{
			Type:      string(ReportKind),
			TestSuite: "__global",
			Blocking:  blocking,
		},
		Definition: def,
	}
}

type SerializedOp struct {
	Test      *TestOp      `yaml:",omitempty"`
	Report    *ReportOp    `yaml:",omitempty"`
	ReportAll *ReportAllOp `yaml:",omitempty"`
}

func SerializeOp0(op Operater) (sop SerializedOp) {
	switch o := op.(type) {
	case *TestOp:
		sop.Test = o
	case *ReportOp:
		sop.Report = o
	case *ReportAllOp:
		sop.ReportAll = o
	default:
		err := fmt.Errorf("unable to serialize operation")
		panic(err)
	}
	return
}

func SerializeOp(op Operater) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	sop := SerializeOp0(op)
	err := enc.Encode(sop)
	if err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}

func DeserializeOp0(sop SerializedOp) (op Operater) {
	if sop.Test != nil {
		return sop.Test
	} else if sop.Report != nil {
		return sop.Report
	} else if sop.ReportAll != nil {
		return sop.ReportAll
	}
	err := fmt.Errorf("unable to deserialize operation")
	panic(err)
}

func DeserializeOp(b []byte) (op Operater, err error) {
	buf := bytes.NewReader(b)
	dec := gob.NewDecoder(buf)
	var sop SerializedOp
	err = dec.Decode(&sop)
	op = DeserializeOp0(sop)
	return
}
