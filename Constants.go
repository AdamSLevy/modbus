package modbus

import (
	"errors"
)

// Mode is the modbus connection mode type.
type Mode byte

// The available modbus connection modes.
const (
	ModeTCP Mode = iota
	ModeRTU
	ModeASCII
)

// DefaultPort is the default port number for Modbus TCP
const (
	DefaultPort = 502
)

// MaxRTUSize MaxASCIISize and MaxTCPSize define the maximum allowable number
// of byes in a single Modbus packet.
const (
	MaxRTUSize   = 512
	MaxASCIISize = 512
	MaxTCPSize   = 260
)

// FunctionCode is the modbus function code type.
type FunctionCode byte

// Modbus Function Codes
const (
	FunctionReadCoils              FunctionCode = 0x01
	FunctionReadDiscreteInputs                  = 0x02
	FunctionReadHoldingRegisters                = 0x03
	FunctionReadInputRegisters                  = 0x04
	FunctionWriteSingleCoil                     = 0x05
	FunctionWriteSingleRegister                 = 0x06
	FunctionWriteMultipleCoils                  = 0x0F
	FunctionWriteMultipleRegisters              = 0x10
	FunctionMaskWriteRegister                   = 0x16
)

// FunctionNames maps function name strings by their Function Code
var FunctionNames = map[FunctionCode]string{
	FunctionReadCoils:              "ReadCoils",
	FunctionReadDiscreteInputs:     "ReadDiscreteInputs",
	FunctionReadHoldingRegisters:   "ReadHoldingRegisters",
	FunctionReadInputRegisters:     "ReadInputRegisters",
	FunctionWriteSingleCoil:        "WriteSingleCoil",
	FunctionWriteSingleRegister:    "WriteSingleRegister",
	FunctionWriteMultipleCoils:     "WriteMultipleCoils",
	FunctionWriteMultipleRegisters: "WriteMultipleRegisters",
	FunctionMaskWriteRegister:      "MaskWriteRegister",
}

// FunctionCodes maps FunctionCodes by their FunctionName, i.e. the inverse of
// the FunctionNames map
var FunctionCodes = map[string]FunctionCode{}

func init() {
	// Initialize FunctionCodes map as the inverse of the FunctionNames map
	for b, s := range FunctionNames {
		FunctionCodes[s] = b
	}
}

// exception indexes into the exceptions map
const (
	exceptionUnspecified                        = 0x00
	exceptionIllegalFunction                    = 0x01
	exceptionDataAddress                        = 0x02
	exceptionDataValue                          = 0x03
	exceptionSlaveDeviceFailure                 = 0x04
	exceptionAcknowledge                        = 0x05
	exceptionSlaveDeviceBusy                    = 0x06
	exceptionMemoryParityError                  = 0x08
	exceptionGatewayPathUnavailable             = 0x0A
	exceptionGatewayTargetDeviceFailedToRespond = 0x0B
	exceptionBadFraming                         = 0xfc // this is not official
	exceptionSlaveIDMismatch                    = 0xfd // this is not official
	exceptionWriteDataMismatch                  = 0xfe // this is not official
	exceptionBadChecksum                        = 0xff // this is not official
)

// exceptions contains a map of common exceptions that may be returned by a
// Packager in the course of sending a Query.
var exceptions = map[uint16]error{
	exceptionUnspecified: errors.New(
		"Modbus Error"),
	exceptionIllegalFunction: errors.New(
		"Modbus Error: Illegal Function (0x01)"),
	exceptionDataAddress: errors.New(
		"Modbus Error: Data Address (0x02)"),
	exceptionDataValue: errors.New(
		"Modbus Error: Data Value (0x03)"),
	exceptionSlaveDeviceFailure: errors.New(
		"Modbus Error: Slave Device Failure (0x04)"),
	exceptionAcknowledge: errors.New(
		"Modbus Error: Acknowledge (0x05)"),
	exceptionSlaveDeviceBusy: errors.New(
		"Modbus Error: Slave Device Busy (0x06)"),
	exceptionMemoryParityError: errors.New(
		"Modbus Error: Memory Parity Error (0x08)"),
	exceptionGatewayPathUnavailable: errors.New(
		"Modbus Error: Gateway Path Unavailable (0x0A)"),
	exceptionGatewayTargetDeviceFailedToRespond: errors.New(
		"Modbus Error: Gateway Target Device Failed to Respond (0x0B)"),
	exceptionBadFraming: errors.New(
		"Modbus Error: Bad Framing"),
	exceptionSlaveIDMismatch: errors.New(
		"Modbus Error: SlaveID mismatch"),
	exceptionWriteDataMismatch: errors.New(
		"Modbus Error: Write data mismatch"),
	exceptionBadChecksum: errors.New(
		"Modbus Error: Bad Checksum"),
}
