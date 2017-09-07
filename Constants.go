// Package modbusclient provides modbus Serial Line/RTU and TCP/IP access
// for client (master) applications to communicate with server (slave)
// devices. This file specifies core definitions and data structures.

package modbus

import (
	"errors"
)

// Mode can be set to ModeTCP, ModeRTU, or ModeASCII
type Mode uint8

// The available modbus connection modes
const (
	ModeTCP Mode = iota
	ModeRTU
	ModeASCII
)

// DefaultPort is the default port number for Modbus TCP
const (
	DefaultPort = 502
)

// MaxRTUSize MaxASCIISize and MaxTCPSize define the maximum allowable
// number of byes in a single Modbus packet.
const (
	MaxRTUSize   = 512
	MaxASCIISize = 512
	MaxTCPSize   = 260
)

// Modbus Function Codes
const (
	FunctionReadCoils                = 0x01
	FunctionReadDiscreteInputs       = 0x02
	FunctionReadHoldingRegisters     = 0x03
	FunctionReadInputRegisters       = 0x04
	FunctionWriteSingleCoil          = 0x05
	FunctionWriteSingleRegister      = 0x06
	FunctionWriteMultipleSingleCoils = 0x0F
	FunctionWriteMultipleRegisters   = 0x10
	FunctionEncapsulatedInterface    = 0x2B
)

const (
	exceptionUnspecified                        = 0x00 // Catch-all for unspecified modbus errors
	exceptionIllegalFunction                    = 0x01
	exceptionDataAddress                        = 0x02
	exceptionDataValue                          = 0x03
	exceptionSlaveDeviceFailure                 = 0x04
	exceptionAcknowledge                        = 0x05
	exceptionSlaveDeviceBusy                    = 0x06
	exceptionMemoryParityError                  = 0x08
	exceptionGatewayPathUnavailable             = 0x0A
	exceptionGatewayTargetDeviceFailedToRespond = 0x0B
	exceptionBadChecksum                        = 0xff // this is not official
)

var exceptions = map[uint16]error{
	exceptionUnspecified:                        errors.New("Modbus Error"),
	exceptionIllegalFunction:                    errors.New("Modbus Error: Illegal Function (0x01)"),
	exceptionDataAddress:                        errors.New("Modbus Error: Data Address (0x02)"),
	exceptionDataValue:                          errors.New("Modbus Error: Data Value (0x03)"),
	exceptionSlaveDeviceFailure:                 errors.New("Modbus Error: Slave Device Failure (0x04)"),
	exceptionAcknowledge:                        errors.New("Modbus Error: Acknowledge (0x05)"),
	exceptionSlaveDeviceBusy:                    errors.New("Modbus Error: Slave Device Busy (0x06)"),
	exceptionMemoryParityError:                  errors.New("Modbus Error: Memory Parity Error (0x08)"),
	exceptionGatewayPathUnavailable:             errors.New("Modbus Error: Gateway Path Unavailable (0x0A)"),
	exceptionGatewayTargetDeviceFailedToRespond: errors.New("Modbus Error: Gateway Target Device Failed to Respond (0x0B)"),
	exceptionBadChecksum:                        errors.New("Modbus Error: Bad Checksum"),
}
