package modbus

import (
	"encoding/binary"
	"errors"
	"github.com/tarm/serial"
	"log"
	"time"
)

// RTUPackager implements the Packager interface for Modbus RTU.
type RTUPackager struct {
	packagerSettings
	*serial.Port
}

// NewRTUPackager returns a new, ready to use RTUPackager with the given
// ConnectionSettings.
func NewRTUPackager(c ConnectionSettings) (*RTUPackager, error) {
	p, err := newSerialPort(c)
	if nil != err {
		return nil, err
	}
	return &RTUPackager{Port: p}, nil
}

func (pkgr *RTUPackager) generateADU(q Query) ([]byte, error) {
	data, err := q.data()
	if err != nil {
		return nil, err
	}

	if q.SlaveID == 0 {
		return nil, errors.New("SlaveID cannot be 0 for Modbus RTU")
	}

	packetLen := len(data) + 4

	packet := make([]byte, packetLen)
	packet[0] = q.SlaveID
	packet[1] = byte(q.FunctionCode)
	bytesUsed := 2

	bytesUsed += copy(packet[bytesUsed:], data)

	// add the crc to the end
	packetCrc := crc(packet[:bytesUsed])
	packet[bytesUsed] = byte(packetCrc & 0xff)
	packet[bytesUsed+1] = byte(packetCrc >> 8)
	bytesUsed += 2

	return packet, nil
}

// Send sends the Query and returns the result or and error code.
func (pkgr *RTUPackager) Send(q Query) ([]byte, error) {
	adu, err := pkgr.generateADU(q)
	if err != nil {
		return nil, err
	}

	if pkgr.Debug {
		log.Printf("Tx: %x\n", adu)
	}

	_, err = pkgr.Write(adu)
	if err != nil {
		return nil, err
	}

	time.Sleep(20 * time.Millisecond)
	response := make([]byte, MaxRTUSize)
	n, rerr := pkgr.Read(response)
	if rerr != nil {
		return nil, rerr
	}

	// Confirm the checksum
	computedCrc := crc(response[:n-2])
	if computedCrc != binary.LittleEndian.Uint16(response[n-2:]) {
		return nil, exceptions[exceptionBadChecksum]
	}
	response = response[:n-2]

	// Check the validity of the response
	if valid, err := q.isValidResponse(response); !valid {
		return nil, err
	}

	// Return only the data payload
	if IsReadFunction(q.FunctionCode) {
		return response[3:], nil
	}

	// Return only the data payload
	return response[2:], nil
}

// crc computes and returns a cyclic redundancy check of the given byte array.
func crc(data []byte) uint16 {
	var crc16 uint16 = 0xffff
	l := len(data)
	for i := 0; i < l; i++ {
		crc16 ^= uint16(data[i])
		for j := 0; j < 8; j++ {
			if crc16&0x0001 > 0 {
				crc16 = (crc16 >> 1) ^ 0xA001
			} else {
				crc16 >>= 1
			}
		}
	}
	return crc16
}
