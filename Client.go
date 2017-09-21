package modbus

import (
	"fmt"
	"sync"
	"time"
)

// ConnectionSettings holds all connection settings. For ModeTCP the Host is
// the FQDN or IP address AND the port number. For ModeRTU and ModeASCII the
// Host string holds the full path to the serial device (Linux) or the name of
// the COM port (Windows) and BaudRate must be specified. The Timeout is
// the response timeout for the the underlying connection.
type ConnectionSettings struct {
	Mode
	Host    string
	Baud    int
	Timeout time.Duration
}

// GetClientHandle returns a new ClientHandle for a client with the given
// ConnectionSettings. ConnectionSettings with the same Host string must also
// match exactly for a new ClientHandle to be returned. The client is shutdown
// after all ClientHandles have been Closed. After a client for a given Host
// has been shutdown it can be reopened with different ConnectionSettings.
func GetClientHandle(cs ConnectionSettings) (ClientHandle, error) {
	once.Do(func() {
		// Set up and start the clientManager singleton
		clntMngr = clientManager{
			closeHandle: make(chan string),
			newClient:   make(chan *clientRequest),
			clients:     make(map[string]*client),
			exit:        make(chan interface{}),
		}
		go clntMngr.requestListener()
	})

	// Send a clientRequest to the clientManager
	req := newClientRequest(cs)
	clntMngr.newClient <- req
	res := <-req.response
	return res.clientHandle, res.Err
}

var clntMngr clientManager
var once sync.Once

type clientManager struct {
	closeHandle chan string
	newClient   chan *clientRequest
	clients     map[string]*client
	exit        chan interface{}
}

func (cm *clientManager) requestListener() {
	for {
		select {
		case clReq := <-cm.newClient:
			ch, err := cm.handleClientRequest(clReq)
			go clReq.sendResponse(ch, err)
		case host := <-cm.closeHandle:
			cl := cm.clients[host]
			cl.numOpenHandles--
			if cl.numOpenHandles == 0 {
				close(cl.queries)
				delete(cm.clients, host)
			}
		case <-cm.exit:
			return
		}
	}
}

func (cm *clientManager) handleClientRequest(
	clReq *clientRequest) (*clientHandle, error) {
	cl, ok := cm.clients[clReq.Host]
	if ok {
		if cl.ConnectionSettings !=
			clReq.ConnectionSettings {
			// Host is in use but other
			// ConnectionSettings details didn't match
			err := fmt.Errorf("Host '%s' is already "+
				"in use with different connection "+
				"settings.", cl.Host)
			return nil, err
		}
		return cl.newClientHandle()
	} else {
		// Set up new client
		cl := newClient(clReq.ConnectionSettings)
		ch, err := cl.start()
		if nil == err {
			cm.clients[cl.Host] = cl
		}
		return ch, err
	}
}

// clientRequests are sent to ClientManager to get access to a Client.
type clientRequest struct {
	ConnectionSettings
	response chan clientResponse
}

type clientResponse struct {
	*clientHandle
	Err error
}

// newClientRequest initializes a new clientRequest with a ClientHandle
// channel.
func newClientRequest(cs ConnectionSettings) *clientRequest {
	return &clientRequest{
		ConnectionSettings: cs,
		response:           make(chan clientResponse),
	}
}

// sendResponse is a convenience function used by clientManager's runtime to
// return a ClientHandle for a clientRequest.
func (req *clientRequest) sendResponse(ch *clientHandle, err error) {
	req.response <- clientResponse{clientHandle: ch, Err: err}
}

// ClientHandle provides a handle for sending Queries to a Client.
type clientHandle struct {
	queryQueue chan query
	response   chan queryResponse
	ConnectionSettings
}

type ClientHandle interface {
	Send(q Query) ([]byte, error)
	Close() error
	GetConnectionSettings() ConnectionSettings
}

// Send sends a Query to the associated Client and returns the response and
// error.
func (ch *clientHandle) Send(q Query) ([]byte, error) {
	if nil == ch.queryQueue {
		return nil, fmt.Errorf("ClientHandle has been closed")
	}
	ch.queryQueue <- query{Query: q, response: ch.response}
	res := <-ch.response
	return res.data, res.err
}

// Close closes the ClientHandle. Once all ClientHandles for a given Client
// have been closed, the Client will shutdown.
func (ch *clientHandle) Close() error {
	if nil == ch.queryQueue {
		return fmt.Errorf("ClientHandle was already closed")
	}
	close(ch.queryQueue)
	close(ch.response)
	ch.queryQueue = nil
	return nil
}

func (ch *clientHandle) GetConnectionSettings() ConnectionSettings {
	return ch.ConnectionSettings
}

// newClient sets up a client with the given ConnectionSettings and returns a
// Client interface for requesting ClientHandles.
func newClient(cs ConnectionSettings) *client {
	return &client{
		ConnectionSettings: cs,
	}
}

// client is the underlying type that implements the Client interface.
type client struct {
	ConnectionSettings
	Packager

	queries        chan query
	numOpenHandles uint
}

// start sets up the appropriate Transporter and Packager for the given
// ConnectionSettings and, if successful, starts the client's queryListener and
// queryQueueChannelMonitor goroutines and returns a new ClientHandle.
func (c *client) start() (*clientHandle, error) {
	p, err := NewPackager(c.ConnectionSettings)
	if nil != err {
		return nil, err
	}
	c.Packager = p
	c.queries = make(chan query)
	go c.queryListener()

	ch, _ := c.newClientHandle()

	return ch, nil
}

func (c *client) newClientHandle() (*clientHandle, error) {
	c.numOpenHandles++
	qq := make(chan query)
	go c.queryForwarder(qq)
	ch := &clientHandle{
		ConnectionSettings: c.ConnectionSettings,
		queryQueue:         qq,
		response:           make(chan queryResponse),
	}
	return ch, nil
}

func (c *client) queryForwarder(qq <-chan query) {
	// This watch group tracks the number of open ClientHandles
	defer func() {
		clntMngr.closeHandle <- c.Host
	}()
	for q := range qq {
		c.queries <- q
	}
}

// queryListener executes Queries sent on the qq and sends queryResponses to
// the Query's Response channel.
func (c *client) queryListener() {
	// Close the Transporter on exit
	defer c.Close()

	// Set up connection for slave
	for qry := range c.queries {
		qry := qry
		time.Sleep(15 * time.Millisecond)
		d, e := c.Send(qry.Query)
		go qry.sendResponse(d, e)
	}
}

// query encapsulates a Query with a queryResponse channel so it can be sent to
// a Client.
type query struct {
	Query
	response chan queryResponse
}

// sendResponse is used by Clients for sending the return queryResponse.
func (q *query) sendResponse(data []byte, err error) {
	q.response <- queryResponse{data: data, err: err}
}

// queryResponse encapsulates the response Data and Err error for a query.
type queryResponse struct {
	data []byte
	err  error
}
