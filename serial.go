package modbus

import (
	"github.com/AdamSLevy/serial"
)

// newSerialPort is used by both the ASCIIPackager and the RTUPackager to set
// up the serial port implementing their Transporter interface.
func newSerialPort(c ConnectionSettings) (*serial.Port, error) {
	conf := &serial.Config{
		Name:        c.Host,
		Baud:        int(c.Baud),
		ReadTimeout: c.Timeout,
	}
	return serial.OpenPort(conf)
}
