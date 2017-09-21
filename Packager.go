package modbus

import "errors"

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
	SetDebug(debug bool)
}

// PackagerSettings holds settings and data that all packagers use.
// Packagers subclass this struct and implement the Packager interface for
// their respective Modbus protocols.
type packagerSettings struct {
	Debug bool
}

func (ps *packagerSettings) SetDebug(debug bool) {
	ps.Debug = debug
}

func NewPackager(cs ConnectionSettings) (Packager, error) {
	switch cs.Mode {
	case ModeTCP:
		return NewTCPPackager(cs)
	case ModeRTU:
		return NewRTUPackager(cs)
	case ModeASCII:
		return NewASCIIPackager(cs)
	default:
		return nil, errors.New("Invalid Mode")
	}
}
