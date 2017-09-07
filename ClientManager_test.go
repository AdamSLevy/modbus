package modbus

import (
	"testing"
	"time"
)

func TestClientManagerInitialization(t *testing.T) {
	if nil != clientManager {
		t.Error("Global clientManager pointer is not nil before initialization.")
	}
	cm := GetClientManager()
	if nil == clientManager {
		t.Error("GetClientManager failed to initialize global clientManager " +
			"pointer")
	}
	if nil == cm {
		t.Error("GetClientManager() returned nil")
	}
	if nil == cm.clients || nil == cm.newConnection || nil == cm.deleteClient {
		t.Error("ClientManager was not properly initialized")
	}
	if cm != GetClientManager() {
		t.Error("GetClientManager() returned two different pointers")
	}
}

func TestClientManagerRequestListener(t *testing.T) {
	cm := GetClientManager()
	req := NewConnectionRequest()

	ch := make(chan interface{})
	go func() {
		cm.SendRequest(req)
		ch <- true
	}()

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Error("ClientManager did not read the ConnectionRequest. SendRequest " +
			"timed out")
	}

	select {
	case res := <-req.Response:
		if nil != res.QueryQueue || nil == res.Err {
			t.Error("ClientManager set up a QueryQueue for an invalid " +
				"ConnectionRequest")
		}
	case <-time.After(2 * time.Second):
		t.Error("ClientManager never sent a response to the ConnectionRequest")
	}

	req.Host = "/dev/pts/3"
	req.Baud = 19200
	req.Mode = ModeASCII
	cm.SendRequest(req)

	var res, res2 *ConnectionResponse
	select {
	case res = <-req.Response:
		if nil == res.QueryQueue || nil != res.Err {
			t.Error("ClientManager set up a QueryQueue for an invalid " +
				"ClientRequest")
		}
	case <-time.After(2 * time.Second):
		t.Error("ClientManager never sent a response to the ClientRequest")
	}

	req2 := NewConnectionRequest()
	req2.ConnectionSettings = req.ConnectionSettings
	cm.SendRequest(req2)
	select {
	case res2 = <-req2.Response:
		if nil == res2.QueryQueue || nil != res2.Err {
			t.Error("ClientManager set up a QueryQueue for an invalid " +
				"ClientRequest")
		}
	case <-time.After(2 * time.Second):
		t.Error("ClientManager never sent a response to the ClientRequest")
	}

	if res.QueryQueue == res2.QueryQueue {
		t.Error("Two separate ClientRequests returned the same QueryQueue")
	}

	_, ok := cm.clients[req.Host]
	if !ok {
		t.Errorf("A Client for Host %v does not exist.", req.Host)
	}

	close(res.QueryQueue)
	close(res2.QueryQueue)

	time.Sleep(1 * time.Millisecond)

	_, ok = cm.clients[req.Host]
	if ok {
		t.Errorf("A Client for Host %v persists after closing all open "+
			"QueryQueues.", req.Host)
	}
}
