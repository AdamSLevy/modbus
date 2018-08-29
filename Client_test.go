package modbus

import (
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	t.Run("GetClientHandle", func(t *testing.T) {
		for _, cs := range testConSettings {
			var validStr string
			if cs.isValid {
				validStr = "valid/"
			} else {
				validStr = "invalid/"
			}
			t.Run(validStr+ModeNames[cs.Mode], func(t *testing.T) {
				testGetClientHandle(t, cs)
			})
		}
	})

	t.Run("Send", runSendTests)

	// Shutdown the clntMngr, this is just for testing purposes to avoid a
	// data race
	clntMngr.exit <- true
	time.Sleep(50 * time.Millisecond)
	if len(clntMngr.clients) > 0 {
		t.Fatal("Clients did not shutdown on close")
	}
}

func testGetClientHandle(t *testing.T, cs testConnectionSettings) {
	t.Parallel()
	done := make(chan interface{})
	var ch ClientHandle
	var err error
	go func() {
		ch, err = GetClientHandle(cs.ConnectionSettings)
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(1000 * time.Millisecond):
		t.Fatal("GetClient timed out")
	}
	if cs.isValid {
		if nil != err {
			t.Fatal(err)
		}
		if ch.GetConnectionSettings() != cs.ConnectionSettings {
			t.Errorf("Incorrect ConnectionSettings, want %v got %v",
				cs.ConnectionSettings, ch.GetConnectionSettings())
		}
		cs := cs.ConnectionSettings
		cs.Timeout += 500
		_, err := GetClientHandle(cs)
		if nil == err {
			t.Error("Altered ConnectionSettings err is nil")
		}
		if ch.Close() != nil {
			t.Error(err)
		}
		if ch.Close() == nil {
			t.Error("Second Close is nil")
		}
		if _, err := ch.Send(testQueries[0].Query); nil == err {
			t.Error("Send after Close returned " +
				"nil error")
		}
	} else {
		if nil == err {
			t.Error("Did not return an error")
		}
	}
}

func testSend(t *testing.T, cs ConnectionSettings, q testQuery) {
	t.Parallel()
	ch, err := GetClientHandle(cs)
	if nil != err {
		t.Fatal(err)
	}
	done := make(chan interface{})
	var data []byte
	go func() {
		data, err = ch.Send(q.Query)
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
	if err := ch.Close(); nil != err {
		t.Error(err)
	}
}

func runSendTests(t *testing.T) {
	for _, cs := range testConSettings {
		if cs.isValid {
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
				testName := ModeNames[cs.Mode] + "/" +
					validStr + "/" +
					fName + "/" +
					q.test
				t.Run(testName, func(t *testing.T) {
					testSend(t, cs.ConnectionSettings, q)
				})
			}
		}
	}
}
