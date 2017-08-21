package modbus

// Transporter is the underlying communication interface and connection. This
// is used to store either a TCP connection or a serial/comm port.
type Transporter interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}
