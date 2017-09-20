package modbus

import (
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientManager(t *testing.T) {
	t.Run("GetClientManager", func(t *testing.T) {
		cm1, err := GetClientManager()
		if nil != err {
			t.Fatal(err)
		}
		cmPtr := clntMngr.Load().(*clientManager)
		if nil == cmPtr.clients ||
			nil == cmPtr.newClient ||
			nil == cmPtr.deleteClient {
			t.Fatal("ClientManager was not properly initialized")
		}
		cm2, _ := GetClientManager()
		if cm1 != cm2 {
			t.Fatal("GetClientManager() returned two different " +
				"pointers")
		}
		cl, err := NewClient(ConnectionSettings{})
		if nil == err {
			t.Error("NewClient did not return an error after GetClientManager")
		}
		if nil != cl {
			t.Error("NewClient returned a Client after GetClientManager")
		}
	})
	t.Run("SetupClient", func(t *testing.T) {
		for _, cs := range testConSettings {
			cs := cs
			var validStr string
			if cs.isValid {
				validStr = "valid/"
			} else {
				validStr = "invalid/"
			}
			t.Run(validStr+ModeNames[cs.Mode], func(t *testing.T) {
				t.Parallel()
				time.Sleep(cs.delay)
				cm, err := GetClientManager()
				if nil != err {
					t.Fatal(err)
				}
				done := make(chan interface{}, 1)
				var ch ClientHandle
				go func() {
					ch, err = cm.SetupClient(cs.ConnectionSettings)
					done <- true
				}()
				select {
				case <-done:
				case <-time.After(500 * time.Millisecond):
					t.Fatal("SetupClient timed out")
				}
				if cs.isValid {
					if nil != err {
						t.Error(err)
					}
					if nil == ch {
						t.Fatal("ClientHandle is nil")
					} else if ch.Close() != nil {
						t.Fatal(err)
					}
				} else {
					if nil == err {
						t.Fatal("Did not return an error")
					}
				}
				if ch.Close() == nil {
					t.Fatal("Second Close is nil")
				}
				if _, err := ch.Send(testQueries[0].Query); nil == err {
					t.Fatal("Send after Close returned " +
						"nil error")
				}

			})
		}
	})

	t.Run("Send", func(t *testing.T) {
		runSendTests(t)
	})
	// Give time for the clients to shutdown
	time.Sleep(50 * time.Millisecond)
	cm, _ := GetClientManager()
	if len(cm.(*clientManager).clients) > 0 {
		log.Println(cm.(*clientManager).clients)
		t.Fatal("Clients did not shutdown on close")
	}
	clntMngr = new(atomic.Value)
	once = new(sync.Once)
	if clntMngr.Load() != nil {
		t.Error("clntMngr not reset")
	}
}

func testSend(t *testing.T, cs ConnectionSettings, q testQuery) {
	t.Parallel()
	cm, err := GetClientManager()
	if nil != err {
		t.Fatal(err)
	} else if nil == cm {
		t.Fatal("ClientManager is nil")
	}
	ch, err := cm.SetupClient(cs)
	if nil != err {
		t.Fatal(err)
	} else if nil == ch {
		t.Fatal("*ClientHandle is nil")
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
	case <-time.After(5000 * time.Millisecond):
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
