package modbus

import (
	"errors"
	"fmt"
	"sync"

	"github.com/goburrow/modbus"
)

// ConnectionManager is a singleton object that keeps track of Connections
// globally for the entire program. Use ConnectionManager() to get a pointer to
// the global ConnectionManager singleton. Connections are hashed on their Host
// field. Connections are set up and accessed by sending ConnectionRequests to
// the ConnectionManager goroutine using SendRequest().
type ConnectionManager struct {
	newConnection chan ConnectionRequest
	connections   map[string]Connection
}

var connectionManager *ConnectionManager
var once sync.Once

// ConnectionManager returns a pointer to the singleton instance of
// ConnectionManager, initializing and starting the ConnectionManager goroutine
// if necessary.
func ConnectionManager() *ConnectionManager {
	once.Do(func() {
		connectionManager = &ConnectionManager{
			newConnection: make(chan ConnectionRequest),
			connections:   make(map[string]Connection),
		}

		go connectionManager.run(connectionManager.newConnection)
	})

	return connectionManager
}

// SendRequest sends a ConnectionRequest to the ConnectionManager runtime. The
// caller should expect a ConnectionResponse on the Response channel.
func (cm *ConnectionManager) SendRequest(req *ConnectionRequest) error {
	if nil == cm.newConnection {
		return errors.New("Uninitialized ConnectionManager")
	}
	go func() {
		cm.newConnection <- req
	}()
	return nil
}

// ConnectionRequests are sent to ConnectionManager's runtime to get a
// ConnectionResponse back on the Response channel. The Connection is set up if
// it does not exist. If a Connection with the same Host already exists, all
// settings must match for a successful ConnectionResponse.
type ConnectionRequest struct {
	Connection
	Response chan *ConnectionResponse
}

// NewConnectionRequest creates a new ConnectionRequest with an initialized
// Response channel. User must then set the Connection settings directly.
func NewConnectionRequest() *ConnectionRequest {
	return &ConnectionRequest{
		Response: make(chan *ConnectionResponse),
	}
}

// sendResponse is a convenience function for sending a ConnectionResponse.
func (req *ConnectionRequest) sendResponse(res *ConnectionResponse) {
	req.Response <- res
}

// ConnectionResponse contains the QueryQueue channel for the Connection
// requested in a ConnectionRequest previously sent to the ConnectionManager.
// The QueryQueue channel can then be used to queue a Query on the Connection
// resource.
type ConnectionResponse struct {
	QueryQueue chan Query
	Err        error
}

// run listens for incoming ConnectionRequests and sends a ConnectionResponse
// to the ConnectionRequest.Response channel. On success, the
// ConnectionResponse has a valid QueryQueue channel for sending queries to the
// requested connection. On failure, ConnectionResponse.Error is set. Failure
// will occur if the connection fails or if a connection for the requested Host
// already exists with different settings. Existing connections can only be
// requested if all settings match exactly.
func (cm *ConnectionManager) run(new <-chan ConnectionRequest) {
	for conReq := range new {
		if nil == conReq.Response {
			continue
		}
		con, ok := cm.connections[conReq.Host]
		if ok {
			if con.Mode == conReq.Mode &&
				con.Baud == conReq.Baud {
				conReq.Response <- &ConnectionResponse{
					QueryQueue: con.queryQueue,
				}
			} else {
				// Host is in use but other
				// connection details didn't match
				err := errors.New(fmt.Sprintf("Host '%s' is already "+
					"in use with different connection settings.",
					con.Host))
				go conReq.sendResponse(&ConnectionResponse{Err: err})
			}
		} else {
			// Set up new connection
			con = conReq.Connection
			err := con.setUpConnection()
			if nil != err {
				go conReq.sendResponse(&ConnectionResponse{Err: err})
				continue
			}
			cm.connections[con.Host] = con
			go conReq.sendResponse(&ConnectionResponse{
				QueryQueue: con.queryQueue,
			})
		}
	}
}

type SlaveId_t uint8
type ModbusMode_t uint8

const (
	MODE_TCP ModbusMode_t = iota
	MODE_RTU
	MODE_ASCII
)

// Connection contains the connection settings, the connection handler, and the
// queryQueue used to listen for queries.
type Connection struct {
	Mode ModbusMode_t
	Host string
	Baud int

	queryQueue chan Query
	handler    modbus.ClientHandler
}

// setUpConnection parses the connection information into a handler, attempts
// the connection, and if successful initializes the queryQueue and starts the
// Connection's goroutine. This is called by ConnectionManager's goroutine to
// set up new Connections.
func (c *Connection) setUpConnection() error {
	switch c.Mode {
	case MODE_TCP:
		handler := modbus.NewTCPClientHandler(c.Host)
		err := handler.Connect()
		if nil != err {
			return err
		}
		c.handler = handler
	case MODE_RTU:
		handler := modbus.NewRTUClientHandler(c.Host)
		handler.BaudRate = c.Baud
		err := handler.Connect()
		if nil != err {
			return err
		}
		c.handler = handler
	case MODE_ASCII:
		handler := modbus.NewASCIIClientHandler(c.Host)
		handler.BaudRate = c.Baud
		err := handler.Connect()
		if nil != err {
			return err
		}
		c.handler = handler
	}
	c.queryQueue = make(chan Query)
	go c.run()
	return nil
}

func (h *modbus.Handler) setSlaveId(id uint8) {
	switch h.(type) {
	case modbus.TCPClientHandler:
		h.SlaveId = id
	case modbus.RTUClientHandler:
		h.SlaveId = id
	case modbus.ASCIIClientHandler:
		h.SlaveId = id
	}
}

// The Connection goroutine that executes Queries sent on the queryQueue and
// sends QueryResponses to the Query's Response channel.
func (con *Connection) run() {
	// Set up client for slave
	c = modbus.NewClient(con.handler)
	for qry := range con.queryQueue {
		id := qry.SlaveId
		con.handler.setSlaveId(id)
		switch qry.FunctionCode {
		case modbus.FuncCodeReadCoils:
			res, err := c.ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeReadDiscreteInputs:
			res, err := c.ReadDiscreteInputs(qry.Address, qry.Quantity)
		case modbus.FuncCodeWriteSingleCoil:
			if 1 != len(qry.Data) {
				err := errors.New("No Query data")
			} else if 0xFF00 != qry.Data[0] &&
				0x0000 != qry.Data[0] {
				err := errors.New("WriteSingleCoil data not well formed")
			} else {
				res, err := c.FuncCodeWriteSingleCoil(qry.Address,
					qry.Data[0])
			}
		case modbus.FuncCodeWriteMultipleCoils:
			if qry.Quantity/16 +  != len(qry.Data) {
				err := errors.New("No Query data")
				break
			}
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeReadInputRegisters:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeReadHoldingRegisters:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeWriteSingleRegister:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeWriteMultipleRegisters:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeReadWriteMultipleRegisters:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeMaskWriteRegister:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		case modbus.FuncCodeReadFIFOQueue:
			res, err := ReadCoils(qry.Address, qry.Quantity)
		}

	}
}

// Query contains the raw information for a Modbus query and the Response
// channel to receive the response data on.
type Query struct {
	SlaveId      SlaveId_t
	FunctionCode uint8
	Address      uint16
	Quantity     uint16
	Data         []byte
	Response     chan *QueryResponse
}

// NewQuery returns a pointer to an initialized Query with a valid Response
// channel for receiving the QueryResponse.
func NewQuery() *Query {
	return &Query{
		Response: make(chan *QueryResponse),
	}
}

// sendResponse is a convenience function for sending a ConnectionResponse.
func (q *Query) sendResponse(res *QueryResponse) {
	q.Response <- res
}

// QueryResponse contains the Data and Err for a previous Query sent to a Connection
type QueryResponse struct {
	Data []byte
	Err  error
}
