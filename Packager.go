package modbus

// Transporter is the underlying communication interface and connection. This
// is used to store either a TCP connection or a serial/comm port.
type Transporter interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}

// Packager generates the raw bytes of a Modbus packet for a given Query.
type Packager interface {
	SetQuery(q Query)
	GetADU() []byte
	GenerateADU() error
	Send() ([]byte, error)
	Transporter
	Querier
}

// PackagerSettings holds settings and data that all packagers must use.
// Packagers subclass this struct and implement the Packager interface for
// their respective Modbus protocols.
type PackagerSettings struct {
	Query
	Debug bool

	adu          []byte
	aduGenerated bool
}

func (pkgr *PackagerSettings) SetQuery(q Query) {
	pkgr.Query = q
	pkgr.aduGenerated = false
}

func (pkgr *PackagerSettings) GetADU() []byte {
	return pkgr.adu
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
