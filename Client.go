package modbus

import (
	"errors"
	"fmt"
	"github.com/tarm/goserial"
	"net"
	"sync"
)

// Connection holds all connection settings. Connections and Clients are
// uniquely identified by their Host string. For ModeTCP this is the FQDN or IP
// address AND the port number. For ModeRTU and ModeASCII the Host string holds
// the full path to the serial device (Linux) or the name of the COM port
// (Windows).
type ConnectionSettings struct {
	Mode
	Host string
	Baud int
}

// Client contains the connection settings, the connection handler, and the
// queryQueue used to listen for queries.
type Client struct {
	ConnectionSettings
	Packager

	mu sync.Mutex
	wg sync.WaitGroup

	queryQueue  chan *Query
	newQQSignal chan interface{}
}

// queryListener executes Queries sent on the queryQueue and sends
// QueryResponses to the Query's Response channel.
func (c *Client) queryListener() {
	defer c.Close()
	// Set up client for slave
	for qry := range c.queryQueue {
		if nil == qry.Response {
			fmt.Println("No Query.Response channel set up")
			continue
		}
		err := c.GeneratePacket(qry)
		if nil != err {
			go qry.sendResponse(&QueryResponse{Err: err})
			continue
		}
		res, err := c.Send()
		if nil != err {
			//Log error
			go qry.sendResponse(&QueryResponse{Err: err})
			continue
		}
		go qry.sendResponse(&QueryResponse{Data: res})
	}
}

// StartClient sets up the appropriate transporter and packager and if
// successful, creates the queryQueue channel and starts the Connection's
// goroutine.
func (c *Client) StartClient() (chan *Query, error) {
	var t Transporter
	switch c.Mode {
	case ModeTCP:
		// make sure the server:port combination resolves to a valid TCP address
		addr, err := net.ResolveTCPAddr("tcp4", c.Host)
		if err != nil {
			return nil, err
		}

		// attempt to connect to the slave device (server)
		t, err = net.DialTCP("tcp", nil, addr)
		if err != nil {
			return nil, err
		}
	case ModeRTU:
		fallthrough
	case ModeASCII:
		conf := &serial.Config{Name: c.Host, Baud: c.Baud}
		var err error
		t, err = serial.OpenPort(conf)
		if nil != err {
			return nil, err
		}
	}
	switch c.Mode {
	case ModeTCP:
		//c.Packager = NewTCPPackager(t)
	case ModeRTU:
		//c.Packager = NewRTUPackager(t)
	case ModeASCII:
		p := NewASCIIPackager(t)
		p.Debug = true
		c.Packager = p
	}

	c.queryQueue = make(chan *Query)
	c.newQQSignal = make(chan interface{})
	go c.queryListener()

	qq, _ := c.newQueryQueue()
	go func() {
		var run = true
		for run {
			c.wg.Wait()
			c.mu.Lock()
			select {
			case <-c.newQQSignal:
				go func() {
					c.newQQSignal <- true
				}()
				c.mu.Unlock()
				continue
			default:
				run = false
			}
		}
		clientManager.deleteClient <- &c.Host
		close(c.queryQueue)
		close(c.newQQSignal)
		c.queryQueue = nil
		c.newQQSignal = nil
		c.mu.Unlock()
	}()
	return qq, nil
}

func (c *Client) queryQueueChannelMonitor() {
	var run = true
	for run {
		// Wait until all QueryQueues have signaled Done()
		c.wg.Wait()
		c.mu.Lock()
		// This is a check for any QueryQueues that may have been created
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
	clientManager.deleteClient <- &c.Host
	close(c.queryQueue)
	close(c.newQQSignal)
	c.queryQueue = nil
	c.newQQSignal = nil
	c.mu.Unlock()
}

// newQueryQueue generates a new QueryQueue channel and a goroutine that
// forwards the queries onto the Client's main internal queryQueue. Each
// goroutine that sends queries to the Client needs their own QueryQueue if
// they are to be allowed to close the channel. Clients with no remaining open
// channels shut themselves down.
func (c *Client) newQueryQueue() (chan *Query, error) {
	if nil == c.queryQueue {
		return nil, errors.New("Client is not running")
	}
	// This watch group tracks the number of open channels
	c.wg.Add(1)
	qq := make(chan *Query)

	// This goroutine sends a blocking newQQSignal which is cleared when
	// the forwarding goroutine exits on channel close. This allows the
	// queryQueueChannelMonitor to avoid a race condition between shutting
	// down the client due to all channels closing and another goroutine,
	// such as the the ClientManager's requestListener, setting up a new
	// QueryQueue channel.
	go func() {
		c.newQQSignal <- true
	}()
	// This goroutine forwards queries from the newly created QueryQueue
	// onto the Client's main internal queryQueue.
	go func() {
		for q := range qq {
			c.queryQueue <- q
		}
		<-c.newQQSignal // Consume newQQSignal before signaling Done()
		c.wg.Done()
	}()
	return qq, nil
}
