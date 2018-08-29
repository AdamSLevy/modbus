package modbus

import (
	"testing"
	"time"
)

func TestPackager(t *testing.T) {
	for _, cs := range testConSettings {
		var validStr string
		if cs.isValid {
			validStr = "valid/"
		} else {
			validStr = "invalid/"
		}
		p, err := NewPackager(cs.ConnectionSettings)
		t.Run(validStr+ModeNames[cs.Mode], func(t *testing.T) {
			p := p
			cs := cs
			if cs.isValid {
				if nil != err {
					t.Fatal(err)
				}
				testPackager(t, p)
				switch cs.Mode {
				case ModeASCII:
					fallthrough
				case ModeRTU:
					q, _ := ReadCoils(0, 0, 1)
					_, err := p.Send(q)
					if nil == err {
						t.Error("SlaveID=0: err is nil")
					}
				}
				if err := p.Close(); nil != err {
					t.Error(err)
				}
			} else {
				if nil == err {
					t.Error("err is nil")
				}
			}
		})
	}
	_, err := NewPackager(ConnectionSettings{Mode: Mode(10)})
	if nil == err {
		t.Error("NewPackager did not return nil for invalid Mode")
	}
}

func testPackager(t *testing.T, p Packager) {
	t.Parallel()
	for _, q := range testQueries {
		var validStr string
		if q.isValid {
			validStr = "valid"
		} else {
			validStr = "invalid"
		}
		fName, ok := FunctionNames[q.FunctionCode]
		if !ok {
			fName = "Invalid FunctionCode"
		}
		testName := validStr + "/" +
			fName + "/" +
			q.test
		t.Run(testName, func(t *testing.T) {
			testPackagerSend(t, p, q)
		})
	}
}

func testPackagerSend(t *testing.T, p Packager, q testQuery) {
	done := make(chan interface{})
	var data []byte
	var err error
	go func() {
		data, err = p.Send(q.Query)
		done <- true
	}()
	var timedOut bool
	select {
	case <-done:
	case <-time.After(1000 * time.Millisecond):
		t.Error("Send timed out")
		timedOut = true
	}
	if !timedOut {
		if q.isValid {
			if nil != err {
				t.Error(err)
			}
			if nil == data {
				t.Error("Response data is nil")
			}
		} else {
			if nil == err {
				t.Error("Error is nil")
			}
			if nil != data {
				t.Error("Response data is not nil")
			}
		}
	}
}
