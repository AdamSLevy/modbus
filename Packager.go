package modbus

// Packager generates the raw bytes of a Modbus packet for a given Query.
type Packager interface {
	GeneratePacket(*Query) error
	Send() ([]byte, error)
	Transporter
}
