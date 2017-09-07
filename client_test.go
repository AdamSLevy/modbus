package modbus

import (
	"testing"
	"time"
)

func TestClientQueryListener(t *testing.T) {
	cm := GetConnectionManager()
	req := NewConnectionRequest()
	req.Host = "/dev/pts/3"
	req.Baud = 19200
	req.Mode = ModeASCII
	cm.SendRequest(req)

	res := <-req.Response
	if nil != res.Err {
		t.Error("ConnectionRequest failed", res.Err)
	}

	qry := NewQuery()

	qry.FunctionCode = FunctionReadCoils
	qry.SlaveID = 1
	qry.Quantity = 1
	ch := make(chan interface{})
	go func() {
		res.QueryQueue <- qry
		ch <- true
	}()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Error("Sending Query timed out")
	}

	select {
	case qryRes := <-qry.Response:
		if nil == qryRes.Data || nil != res.Err {
			t.Error("Query failed")
		}
	case <-time.After(2 * time.Second):
		t.Error("Awaiting QueryResponse timed out.")
	}
}
