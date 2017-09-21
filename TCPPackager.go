package modbus

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"
)

// TCPPackager implements the Packager interface for Modbus TCP.
type TCPPackager struct {
	packagerSettings
	net.Conn

	transactionID uint16
	timeout       time.Duration
}

// NewTCPPackager returns a new, ready to use TCPPackager with the given
// ConnectionSettings.
func NewTCPPackager(c ConnectionSettings) (*TCPPackager, error) {
	addr, err := net.ResolveTCPAddr("tcp", c.Host)
	// attempt to connect to the slave device (server)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	conn.SetKeepAlive(true)
	return &TCPPackager{
		Conn:    conn,
		timeout: c.Timeout,
	}, nil
}

func (pkgr *TCPPackager) generateADU(q Query) ([]byte, error) {
	data, err := q.data()
	if err != nil {
		return nil, err
	}

	packetLen := len(data) + 8
	packet := make([]byte, packetLen)
	packet[0] = byte(pkgr.transactionID >> 8)   // Transaction ID (High Byte)
	packet[1] = byte(pkgr.transactionID & 0xff) //                (Low Byte)
	packet[2] = 0x00                            // Protocol ID (2 bytes) -- always 00
	packet[3] = 0x00
	packet[4] = byte((len(data) + 2) >> 8)   // Length of remaining packet (High Byte)
	packet[5] = byte((len(data) + 2) & 0xff) // (Low Byte)

	packet[6] = q.SlaveID
	packet[7] = byte(q.FunctionCode)
	copy(packet[8:], data)

	return packet, nil
}

// Send sends the Query and returns the result or and error code.
func (pkgr *TCPPackager) Send(q Query) ([]byte, error) {
	adu, err := pkgr.generateADU(q)
	if err != nil {
		return nil, err
	}

	defer func() { pkgr.transactionID++ }()
	if pkgr.Debug {
		log.Printf("Tx: %x\n", adu)
	}

	pkgr.SetDeadline(time.Now().Add(pkgr.timeout))

	_, err = pkgr.Write(adu)
	if err != nil {
		return nil, err
	}

	pkgr.SetDeadline(time.Now().Add(pkgr.timeout))
	response := make([]byte, MaxTCPSize)
	n, err := pkgr.Read(response)
	if err != nil {
		return nil, err
	}

	// Check for matching transactionID
	if binary.BigEndian.Uint16(response[0:2]) != pkgr.transactionID {
		return nil, errors.New("Mismatched transactionID")
	}

	response = response[6:n]
	// Check the validity of the response
	if valid, err := q.isValidResponse(response); !valid {
		return nil, err
	}

	if isReadFunction(q.FunctionCode) {
		return response[3:], nil
	}
	// return only the number of bytes read
	return response[2:], nil
}
