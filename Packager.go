package modbus

import (
	"fmt"
	"github.com/tarm/serial"
)

// Transporter is the underlying communication interface and connection. This
// is used to store either a TCP connection or a serial/comm port.
type Transporter interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}

// Packager generates the raw bytes of a Modbus packet for a given Query,
// transmits the Query on the underlying Transporter interface, and waits and
// returns the response data. A Packager is implemented for the three modbus
// Modes: ASCIIPackager, RTUPackager and TCPPackager.
type Packager interface {
	Send(q Query) ([]byte, error)
	Transporter
}

// PackagerSettings holds settings and data that all packagers use.
// Packagers subclass this struct and implement the Packager interface for
// their respective Modbus protocols.
type packagerSettings struct {
	Debug bool
}

// isValidResponse is used by Packagers to validate the response data against a
// given Query.
func isValidResponse(q Query, response []byte) (bool, error) {
	if nil == response || len(response) == 0 {
		return false, fmt.Errorf("Empty response")
	}
	// check the validity of the response
	if response[0] != q.SlaveID {
		return false, exceptions[exceptionSlaveIDMismatch]
	}

	if FunctionCode(response[1]) != q.FunctionCode {
		if FunctionCode(response[1]&0x7f) == q.FunctionCode {
			switch response[2] {
			case exceptionIllegalFunction:
				return false, exceptions[exceptionIllegalFunction]
			case exceptionDataAddress:
				return false, exceptions[exceptionDataAddress]
			case exceptionDataValue:
				return false, exceptions[exceptionDataValue]
			case exceptionSlaveDeviceFailure:
				return false, exceptions[exceptionSlaveDeviceFailure]
			}
		}
		return false, exceptions[exceptionUnspecified]
	}

	if IsWriteFunction(q.FunctionCode) {
		data, _ := q.data()
		for i := 0; i < 4; i++ {
			if i >= len(response) || data[i] != response[2+i] {
				return false, exceptions[exceptionWriteDataMismatch]
			}
		}
	}
	return true, nil
}

// newSerialPort is used by both the ASCIIPackager and the RTUPackager to set
// up the serial port implementing their Transporter interface.
func newSerialPort(c ConnectionSettings) (*serial.Port, error) {
	conf := &serial.Config{
		Name:        c.Host,
		Baud:        c.Baud,
		ReadTimeout: c.Timeout,
	}
	return serial.OpenPort(conf)
}
