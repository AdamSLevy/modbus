package modbus

import (
	"encoding/binary"
	"errors"
	"github.com/tarm/serial"
	"log"
)

type RTUPackager struct {
	packagerSettings
	*serial.Port
}

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
	if packetLen > MaxRTUSize {
		return nil, errors.New("Query Data is too long")
	}

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

	// Check the validity of the response
	if valid, err := isValidResponse(q, response); !valid {
		return nil, err
	}

	// Return only the data payload
	if IsReadFunction(q.FunctionCode) {
		return response[3 : n-2], nil
	}

	// Return only the data payload
	return response[2 : n-2], nil
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
