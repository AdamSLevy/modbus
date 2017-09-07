package modbus

// Packager generates the raw bytes of a Modbus packet for a given Query.
type Packager interface {
	GeneratePacket(*Query) error
	Send() ([]byte, error)
	Transporter
}

// PackagerSettings holds settings and data that all packagers must use.
// Packagers subclass this struct and implement the Packager interface for
// their respective Modbus protocols.
type PackagerSettings struct {
	Transporter
	*Query

	TimeoutInMilliseconds int
	Validate              bool
	Debug                 bool

	pkt    []byte
	pktLen int
}

func (pkgr *PackagerSettings) isValidResponse(response []byte) (bool, error) {
	// check the validity of the response
	if response[0] != pkgr.SlaveID || response[1] != pkgr.FunctionCode {
		if response[0] == pkgr.SlaveID &&
			(response[1]&0x7f) == pkgr.FunctionCode {
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
	return true, nil
}
