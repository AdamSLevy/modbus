package modbus

import (
	"testing"
	"time"
)

func TestConnectionManager(t *testing.T) {
	t.Run("Singleton initialization", func(t *testing.T) {
		cm := GetConnectionManager()
		if nil == cm {
			t.Error("GetConnectionManager() returned nil")
		}

		if cm != GetConnectionManager() {
			t.Error("GetConnectionManager() returned two different pointers")
		}

		if nil == cm.clients ||
			nil == cm.newConnection ||
			nil == cm.deleteClient {
			t.Error("ConnectionManager was not properly initialized")
		}
	})

	req := NewConnectionRequest()
	t.Run("Reject invalid ConnectionRequest", func(t *testing.T) {
		cm := GetConnectionManager()

		ch := make(chan interface{})
		go func() {
			cm.SendRequest(req)
			ch <- true
		}()

		select {
		case <-ch:
		case <-time.After(500 * time.Millisecond):
			t.Error("ConnectionManager did not read the ConnectionRequest. " +
				"SendRequest timed out")
		}

		select {
		case res := <-req.Response:
			if nil != res.QueryQueue || nil == res.Err {
				t.Error("Invalid ConnectionRequest did not return an " +
					"error")
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("ConnectionRequest response timeout")
		}
	})
}
