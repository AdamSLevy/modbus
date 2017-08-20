package modbus

import (
	"github.com/tarm/goserial"
	"net"

	"errors"
)

type SlaveId_t uint8
type ModbusMode_t uint8

const (
	MODE_TCP ModbusMode_t = iota
	MODE_RTU
	MODE_ASCII
)

type Transporter interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}

type Packager interface {
}

// Client contains the connection settings, the connection handler, and the
// queryQueue used to listen for queries.
type Client struct {
	Mode ModbusMode_t
	Host string
	Baud int

	queryQueue  chan *Query
	transporter Transporter
	packager    Packager
}

// queryListener executes Queries sent on the queryQueue and sends
// QueryResponses to the Query's Response channel.
func (c *Client) queryListener() {
	// Set up client for slave
	for qry := range c.queryQueue {
		qry.sendResponse(&QueryResponse{
			Err: errors.New("Not yet implemented"),
		})
	}
}

// startClient sets up the appropriate communication interface and if
// successful, creates the queryQueue channel and starts the Connection's
// goroutine.
func (c *Client) startClient() error {
	switch c.Mode {
	case MODE_TCP:
		// make sure the server:port combination resolves to a valid TCP address
		addr, err := net.ResolveTCPAddr("tcp4", c.Host)
		if err != nil {
			return err
		}

		// attempt to connect to the slave device (server)
		t, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			return err
		}
		c.transporter = t
	case MODE_RTU:
		fallthrough
	case MODE_ASCII:
		conf := &serial.Config{Name: con.Host, Baud: con.Baud}
		t, err := serial.OpenPort(conf)
		if nil != err {
			return err
		}
		c.transporter = t
	}

	c.queryQueue = make(chan *Query)
	go c.queryListener()
	return nil
}
