package modbus

import (
	"testing"
	"time"
)

func TestClientQueryListener(t *testing.T) {
	cm := GetClientManager()
	req := NewConnectionRequest()
	req.Host = "/dev/pts/3"
	req.Baud = 19200
	req.Mode = ModbusModeASCII
	cm.SendRequest(req)

	res := <-req.Response
	if nil != res.Err {
		t.Error("ClientRequest failed", res.Err)
	}

	qry := NewQuery()

	qry.FunctionCode = FUNCTION_READ_COILS
	qry.SlaveId = 1
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
