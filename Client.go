package modbus

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ClientHandle provides a handle for sending Queries to a Client.
type ClientHandle struct {
	queryQueue chan query
	response   chan queryResponse
	ConnectionSettings
}

// Send sends a Query to the associated Client and returns the response and
// error.
func (ch *ClientHandle) Send(q Query) ([]byte, error) {
	if ch.queryQueue == nil {
		return nil, fmt.Errorf("ClientHandle has been closed")
	}
	if ch.response == nil {
		ch.response = make(chan queryResponse)
	}
	ch.queryQueue <- query{Query: q, response: ch.response}
	res := <-ch.response
	return res.data, res.err
}

// Close closes the ClientHandle. Once all ClientHandles for a given Client
// have been closed, the Client will shutdown.
func (ch *ClientHandle) Close() error {
	if ch.queryQueue == nil {
		return fmt.Errorf("ClientHandle was already closed")
	}
	close(ch.queryQueue)
	close(ch.response)
	ch.queryQueue = nil
	return nil
}

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

// Client is the abstract interface to a client. To start the client and send
// Queries to it, request a *ClientHandle with NewClientHandle. The underlying
// client has a queryListener goroutine that listens for Queries and serializes
// access to its Packager. Clients start when the first ClientHandle is
// requested with Client.GetClientHandle. Clients shutdown and all associated
// goroutines exit when all ClientHandles have been closed with
// ClientHandle.Close(). Clients can be created with NewClient OR ClientHandles
// can be requested from ClientManager.SetupClient, but not both. Using clients
// created with NewClient and using the ClientManager are mutually exclusive
// modes of operation for this library.
type Client interface {
	NewClientHandle() (*ClientHandle, error)
}

// NewClient sets up a client with the given ConnectionSettings and returns a
// Client interface for requesting ClientHandles. Clients created using
// NewClient are not managed by the ClientManager. If you use the ClientManager
// do not use NewClient and instead use ClientManager.SetupClient().
func NewClient(cs ConnectionSettings) (Client, error) {
	switch clntMngr.Load().(type) {
	case *clientManager:
		return nil, fmt.Errorf("ClientManager is already in use. " +
			"Use ClientManager.SetupClient instead of NewClient.")
	}
	atomic.StoreUint32(&unmanagedClients, 1)

	return &client{
		ConnectionSettings: cs,
		isManagedClient:    false,
		newQQSignal:        make(chan interface{}),
	}, nil
}

// newClient sets up a client with the given ConnectionSettings and returns a
// Client interface for requesting ClientHandles.
func newClient(cs ConnectionSettings) *client {
	return &client{
		ConnectionSettings: cs,
		isManagedClient:    true,
		newQQSignal:        make(chan interface{}),
	}
}

// client is the underlying type that implements the Client interface.
type client struct {
	isManagedClient bool
	ConnectionSettings
	Packager

	mu sync.Mutex
	wg sync.WaitGroup

	qq          chan query
	newQQSignal chan interface{}
}

// NewClientHandle starts the client if it isn't already running and then, if
// successful, returns a new ClientHandle and starts a goroutine that forwards
// the queries sent by that ClientHandle onto the client's main internal query
// channel.
func (c *client) NewClientHandle() (*ClientHandle, error) {
	if c.qq == nil {
		return c.start()
	}
	// This watch group tracks the number of open ClientHandles
	c.wg.Add(1)
	qq := make(chan query)

	// Send a blocking newQQSignal to be cleared when the forwarding
	// goroutine exits on channel close. This allows the
	// queryQueueChannelMonitor to avoid a race condition between shutting
	// down the connection due to all channels closing and another
	// goroutine, such as the the ClientManager's requestListener, setting
	// up a new Query channel.
	go func() {
		c.newQQSignal <- true
	}()
	// Forward queries from the newly created QueryQueue onto the
	// connection's main internal qq.
	go func() {
		for q := range qq {
			//log.Println(q)
			c.qq <- q
		}
		<-c.newQQSignal // Consume newQQSignal before signaling Done()
		c.wg.Done()
	}()
	return &ClientHandle{
		ConnectionSettings: c.ConnectionSettings,
		queryQueue:         qq,
		response:           make(chan queryResponse),
	}, nil

}

// start sets up the appropriate Transporter and Packager for the given
// ConnectionSettings and, if successful, starts the client's queryListener and
// queryQueueChannelMonitor goroutines and returns a new *ClientHandle.
func (c *client) start() (*ClientHandle, error) {
	switch c.Mode {
	case ModeTCP:
		p, err := NewTCPPackager(c.ConnectionSettings)
		if nil != err {
			return nil, err
		}
		c.Packager = p
	case ModeRTU:
		p, err := NewRTUPackager(c.ConnectionSettings)
		if nil != err {
			return nil, err
		}
		c.Packager = p
	case ModeASCII:
		p, err := NewASCIIPackager(c.ConnectionSettings)
		if nil != err {
			return nil, err
		}
		c.Packager = p
	}
	c.qq = make(chan query)
	go c.queryListener()

	ch, _ := c.NewClientHandle()
	go c.queryQueueChannelMonitor()

	return ch, nil
}

// queryListener executes Queries sent on the qq and sends queryResponses to
// the Query's Response channel.
func (c *client) queryListener() {
	// Close the Transporter on exit
	defer c.Close()

	// Set up connection for slave
	for qry := range c.qq {
		qry := qry
		if nil == qry.response {
			log.Println("No Query.Response channel set up")
			continue
		}
		go qry.sendResponse(c.Send(qry.Query))
	}
}

// queryQueueChannelMonitor waits for all query forwarding goroutines to exit
// due to their respective ClientHandles being closed. After all ClientHandles
// are closed this goroutine shutsdown the queryListener by closing the
// client.qq query channel.
func (c *client) queryQueueChannelMonitor() {
	var run = true
	for run {
		// Wait until all queryQueue channels have signaled Done()
		c.wg.Wait()
		c.mu.Lock()
		// This is a check for any queryQueue channels that may have been created
		// between Wait() returning and acquiring the Lock().
		select {
		case <-c.newQQSignal:
			// Relaunch the goroutine holding the blocking newQQSignal signal
			go func() {
				c.newQQSignal <- true
			}()
			c.mu.Unlock()
			continue
		default:
			run = false
		}
	}
	if c.isManagedClient {
		// Let the ClientManager know that this client has shutdown
		clntMngr.Load().(*clientManager).deleteClient <- &c.Host
	}
	close(c.qq)
	c.qq = nil
	c.mu.Unlock()
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
