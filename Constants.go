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

// ModeNames maps Mode to a string description
var ModeNames = map[Mode]string{
	ModeTCP:   "TCP",
	ModeRTU:   "RTU",
	ModeASCII: "ASCII",
}

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
	// Official Modbus Exceptions
	exceptionUnknown                            = 0x00
	exceptionIllegalFunction                    = 0x01
	exceptionDataAddress                        = 0x02
	exceptionDataValue                          = 0x03
	exceptionSlaveDeviceFailure                 = 0x04
	exceptionAcknowledge                        = 0x05
	exceptionSlaveDeviceBusy                    = 0x06
	exceptionMemoryParityError                  = 0x08
	exceptionGatewayPathUnavailable             = 0x0A
	exceptionGatewayTargetDeviceFailedToRespond = 0x0B

	// Unofficial exceptions
	exceptionEmptyResponse          = 0xf9
	exceptionBadResponseLength      = 0xfa
	exceptionBadFraming             = 0xfb
	exceptionSlaveIDMismatch        = 0xfc
	exceptionWriteDataMismatch      = 0xfd
	exceptionResponseLengthMismatch = 0xfe
	exceptionBadChecksum            = 0xff
)

// exceptions contains a map of common exceptions that may be returned by a
// Packager in the course of sending a Query.
var exceptions = map[uint16]error{
	exceptionUnknown: errors.New(
		"Modbus Error: Unknown"),
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

	exceptionEmptyResponse: errors.New(
		"Response Error: Empty response"),
	exceptionBadResponseLength: errors.New(
		"Response Error: Bad response length"),
	exceptionBadFraming: errors.New(
		"Response Error: Bad Framing"),
	exceptionSlaveIDMismatch: errors.New(
		"Response Error: SlaveID mismatch"),
	exceptionWriteDataMismatch: errors.New(
		"Response Error: Write data mismatch"),
	exceptionResponseLengthMismatch: errors.New(
		"Response Error: Response length mismatch"),
	exceptionBadChecksum: errors.New(
		"Response Error: Bad Checksum"),
}
