package modbus

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"

	"github.com/tarm/serial"
)

// ASCIIPackager implements the Packager interface for Modbus ASCII.
type ASCIIPackager struct {
	packagerSettings
	*serial.Port
}

// NewASCIIPackager returns a new, ready to use ASCIIPackager with the given
// ConnectionSettings.
func NewASCIIPackager(c ConnectionSettings) (*ASCIIPackager, error) {
	p, err := newSerialPort(c)
	if nil != err {
		return nil, err
	}
	return &ASCIIPackager{
		Port: p,
		packagerSettings: packagerSettings{
			Debug: c.Debug,
		},
	}, nil
}

func (pkgr *ASCIIPackager) generateADU(q Query) ([]byte, error) {
	data, err := q.data()
	if err != nil {
		return nil, err
	}

	if q.SlaveID == 0 {
		return nil, errors.New("SlaveID cannot be 0 for Modbus ASCII")
	}

	packetLen := 2
	packetLen += len(data) + 1
	rawPkt := make([]byte, packetLen)
	rawPkt[0] = q.SlaveID
	rawPkt[1] = byte(q.FunctionCode)
	bytesUsed := 2

	bytesUsed += copy(rawPkt[bytesUsed:], data)

	// add the lrc to the end
	pktLrc := lrc(rawPkt[:bytesUsed])
	rawPkt[bytesUsed] = byte(pktLrc)
	bytesUsed++

	// Convert raw bytes to ASCII packet
	asciiPkt := make([]byte, bytesUsed*2+3)
	hex.Encode(asciiPkt[1:], rawPkt)

	asciiBytesUsed := bytesUsed*2 + 1

	// Frame the packet
	asciiPkt[0] = ':'                 // 0x3A
	asciiPkt[asciiBytesUsed] = '\r'   // CR 0x0D
	asciiPkt[asciiBytesUsed+1] = '\n' // LF 0x0A

	return bytes.ToUpper(asciiPkt), nil
}

// Send sends the Query and returns the result or and error code.
func (pkgr *ASCIIPackager) Send(q Query) ([]byte, error) {
	adu, err := pkgr.generateADU(q)
	if err != nil {
		return nil, err
	}

	if pkgr.Debug {
		log.Printf("Tx: %x\n", adu)
		log.Printf("Tx: %s\n", adu)
	}

	_, err = pkgr.Write(adu)
	if err != nil {
		return nil, err
	}

	asciiResponse := make([]byte, MaxASCIISize)
	asciiN, rerr := pkgr.Read(asciiResponse)
	if rerr != nil {
		return nil, rerr
	}

	if pkgr.Debug {
		log.Printf("Rx Full: %x\n", asciiResponse)
	}

	// Check the framing of the response
	if asciiResponse[0] != ':' ||
		asciiResponse[asciiN-2] != '\r' ||
		asciiResponse[asciiN-1] != '\n' {
		return nil, exceptions[exceptionBadFraming]
	}

	// Convert to raw bytes
	rawN := (asciiN - 3) / 2
	response := make([]byte, rawN)
	hex.Decode(response, asciiResponse[1:asciiN-2])

	// Confirm the checksum
	responseLrc := lrc(response[:rawN-1])
	if response[rawN-1] != responseLrc {
		return nil, exceptions[exceptionBadChecksum]
	}

	response = response[:rawN-1]

	if pkgr.Debug {
		log.Printf("Rx: %x\n", response)
	}

	// Check the validity of the response
	if valid, err := q.isValidResponse(response); !valid {
		return nil, err
	}

	// Return only the data payload
	if isReadFunction(q.FunctionCode) {
		return response[3:], nil
	}
	return response[2:], nil
}

// Modbus ASCII uses Longitudinal Redundancy Check. lrc computes and returns
// the 2's compliment (-) of the sum of the given byte array modulo 256
func lrc(data []byte) uint8 {
	var sum uint8
	var lrc8 uint8
	for _, b := range data {
		sum += b
	}
	lrc8 = uint8(-int8(sum))
	return lrc8
}
