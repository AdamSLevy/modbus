package modbus

import (
	"sync/atomic"
	"testing"
	//"time"
)

func TestClient(t *testing.T) {
	t.Run("NewClient", func(t *testing.T) {
		for _, cs := range testConSettings {
			cs := cs
			var validStr string
			if cs.isValid {
				validStr = "valid/"
			} else {
				validStr = "invalid/"
			}
			t.Run(validStr+ModeNames[cs.Mode], func(t *testing.T) {
				cl, err := NewClient(cs.ConnectionSettings)
				if nil != err {
					t.Error(err)
				}
				if nil == cl {
					t.Fatal("NewClient failed to return a valid Client")
				}
				ch, err := cl.NewClientHandle()
				if cs.isValid {
					if nil != err {
						t.Fatal(err)
					} else if nil == ch {
						t.Fatal("ClientHandle is nil")
					}
					if ch.Close() != nil {
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
		if 1 != atomic.LoadUint32(&unmanagedClients) {
			t.Error("unmanagedClients not set to 1 after calling NewClient")
		}
		cm, err := GetClientManager()
		if nil == err {
			t.Error("Failed to return an error after a call to NewClient")
		}
		if nil != cm {
			t.Error("Returned ClientManager after a call to NewClient")
		}
		atomic.StoreUint32(&unmanagedClients, 0)
	})
}
