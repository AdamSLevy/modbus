package modbus

import (
	//"errors"
	"encoding/binary"
	"fmt"
)

// Querier defines the Modbus functions implemented by Query. These functions
// instantiate a valid Query that can be successfully passed to a Packager for
// execution.
type Querier interface {
	ReadFunction(fCode byte, address, quantity uint16) (bool, error)
	WriteSingleFunction(fCode byte, address, value uint16) (bool, error)
	WriteMultipleFunction(fCode byte, address, quantity uint16,
		value []byte) (bool, error)

	ReadCoils(address, quantity uint16) (bool, error)
	ReadDiscreteInputs(address, quantity uint16) (bool, error)
	ReadInputRegisters(address, quantity uint16) (bool, error)
	ReadHoldingRegisters(address, quantity uint16) (bool, error)
	WriteSingleCoil(address uint16, value bool) (bool, error)
	WriteSingleRegister(address, value uint16) (bool, error)
	WriteMultipleCoils(address, quantity uint16, value []byte) (bool, error)
	WriteMultipleRegisters(address, quantity uint16, value []byte) (bool, error)
	MaskWriteRegister(address, andMask, orMask uint16) (bool, error)
}

// Query contains the necessary data for a Packager to construct and execute a
// Modbus query. Queries implement the Querier interface
type Query struct {
	SlaveID      byte
	FunctionCode byte
	Address      uint16
	Quantity     uint16
	Data         []byte

	Response chan QueryResponse
}

// NewQuery returns a pointer to an initialized Query with a valid Response
// channel for receiving the QueryResponse.
func NewQuery() *Query {
	return &Query{Response: make(chan QueryResponse)}
}

// sendResponse is a convenience function used by Packagers for sending a the
// return QueryResponse.
func (q *Query) sendResponse(data []byte, err error) {
	q.Response <- QueryResponse{Data: data, Err: err}
}

// ReadFunction sets up the Query for a ReadFunction query.
//  Function code         : 1 byte
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
func (q *Query) ReadFunction(fCode byte, address, quantity uint16) (bool, error) {
	q.FunctionCode = fCode
	q.Address = address
	q.Quantity = quantity
	q.Data = dataBlock(address, quantity)

	if !q.IsRead() {
		return false, fmt.Errorf("Not a valid read function code")
	}
	return q.IsValid()
}

// Request:
//  Function code         : 1 byte
//  Starting address      : 2 bytes
//  Register value 	  : 2 bytes
func (q *Query) WriteSingleFunction(fCode byte, address, value uint16) (bool, error) {
	q.FunctionCode = fCode
	q.Address = address
	q.Quantity = 0
	q.Data = dataBlock(address, value)

	if !q.IsWrite() {
		return false, fmt.Errorf("Not a valid write function code")
	}
	return q.IsValid()
}

// Request:
//  Function code         : 1 byte
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
//  Byte count            : 1 byte
//  Outputs value         : N* bytes
func (q *Query) WriteMultipleFunction(fCode byte, address, quantity uint16, value []byte) (bool, error) {
	q.FunctionCode = fCode
	q.Address = address
	q.Quantity = quantity
	q.Data = dataBlockSuffix(value, address, quantity)

	if !q.IsWrite() {
		return false, fmt.Errorf("Not a valid write function code")
	}
	return q.IsValid()
}

// Request:
//  Function code         : 1 byte (0x01)
//  Starting address      : 2 bytes
//  Quantity of coils     : 2 bytes
// Response:
//  Function code         : 1 byte (0x01)
//  Byte count            : 1 byte
//  Coil status           : N* bytes (=N or N+1)
func (q *Query) ReadCoils(address, quantity uint16) (bool, error) {
	return q.ReadFunction(FunctionReadCoils, address, quantity)
}

// Request:
//  Function code         : 1 byte (0x02)
//  Starting address      : 2 bytes
//  Quantity of inputs    : 2 bytes
// Response:
//  Function code         : 1 byte (0x02)
//  Byte count            : 1 byte
//  Input status          : N* bytes (=N or N+1)
func (q *Query) ReadDiscreteInputs(address, quantity uint16) (bool, error) {
	return q.ReadFunction(FunctionReadDiscreteInputs, address, quantity)
}

// Request:
//  Function code         : 1 byte (0x03)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x03)
//  Byte count            : 1 byte
//  Register value        : Nx2 bytes
func (q *Query) ReadHoldingRegisters(address, quantity uint16) (bool, error) {
	return q.ReadFunction(FunctionReadHoldingRegisters, address, quantity)
}

// Request:
//  Function code         : 1 byte (0x04)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
// Response:
//  Function code         : 1 byte (0x04)
//  Byte count            : 1 byte
//  Input registers       : N bytes
func (q *Query) ReadInputRegisters(address, quantity uint16) (bool, error) {
	return q.ReadFunction(FunctionReadInputRegisters, address, quantity)
}

// Request:
//  Function code         : 1 byte (0x05)
//  Output address        : 2 bytes
//  Output value          : 2 bytes
// Response:
//  Function code         : 1 byte (0x05)
//  Output address        : 2 bytes
//  Output value          : 2 bytes
func (q *Query) WriteSingleCoil(address uint16, value bool) (bool, error) {
	var valueData uint16
	if value {
		valueData = 0xFF00
	}
	return q.WriteSingleFunction(FunctionWriteSingleCoil, address, valueData)
}

// Request:
//  Function code         : 1 byte (0x06)
//  Register address      : 2 bytes
//  Register value        : 2 bytes
// Response:
//  Function code         : 1 byte (0x06)
//  Register address      : 2 bytes
//  Register value        : 2 bytes
func (q *Query) WriteSingleRegister(address, value uint16) (bool, error) {
	return q.WriteSingleFunction(FunctionWriteSingleRegister, address, value)
}

// Request:
//  Function code         : 1 byte (0x0F)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Outputs value         : N* bytes
// Response:
//  Function code         : 1 byte (0x0F)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
func (q *Query) WriteMultipleCoils(address, quantity uint16, value []byte) (bool, error) {
	return q.WriteMultipleFunction(FunctionWriteMultipleCoils,
		address, quantity, value)
}

// Request:
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of outputs   : 2 bytes
//  Byte count            : 1 byte
//  Registers value       : N* bytes
// Response:
//  Function code         : 1 byte (0x10)
//  Starting address      : 2 bytes
//  Quantity of registers : 2 bytes
func (q *Query) WriteMultipleRegisters(address,
	quantity uint16, value []byte) (bool, error) {
	return q.WriteMultipleFunction(FunctionWriteMultipleRegisters,
		address, quantity, value)
}

// Request:
//  Function code         : 1 byte (0x16)
//  Reference address     : 2 bytes
//  AND-mask              : 2 bytes
//  OR-mask               : 2 bytes
// Response:
//  Function code         : 1 byte (0x16)
//  Reference address     : 2 bytes
//  AND-mask              : 2 bytes
//  OR-mask               : 2 bytes
func (q *Query) MaskWriteRegister(address, andMask, orMask uint16) (bool, error) {
	q.FunctionCode = FunctionMaskWriteRegister
	q.Address = address
	q.Quantity = 0
	q.Data = dataBlock(address, andMask, orMask)

	if !q.IsWrite() {
		return false, fmt.Errorf("Not a valid write function code")
	}
	return q.IsValid()
}

// dataBlock creates a sequence of uint16 data.
func dataBlock(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// dataBlockSuffix creates a sequence of uint16 data and append the suffix plus its length.
func dataBlockSuffix(suffix []byte, value ...uint16) []byte {
	length := 2 * len(value)
	data := make([]byte, length+1+len(suffix))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	data[length] = uint8(len(suffix))
	copy(data[length+1:], suffix)
	return data
}

// QueryResponse contains the Data and Err for a previous Query sent to a Connection
type QueryResponse struct {
	Data []byte
	Err  error
}

// IsValid returns a bool representing whether the Query is well constructed
// with a supported FunctionCode. If the query is invalid, IsValid returns
// false, and an error describing the reason for not passing.
func (q *Query) IsValid() (bool, error) {
	errString, _ := FunctionNames[q.FunctionCode]
	switch q.FunctionCode {
	case FunctionReadCoils:
		if q.Quantity == 0 || q.Quantity > 2000 {
			return false, fmt.Errorf("%v: Quantity out of range: %v",
				errString, q.Quantity)
		}
	case FunctionReadDiscreteInputs:
		if q.Quantity == 0 || q.Quantity > 2000 {
			return false, fmt.Errorf("%v: Quantity out of range: %v",
				errString, q.Quantity)
		}
	case FunctionReadHoldingRegisters:
		if q.Quantity == 0 || q.Quantity > 125 {
			return false, fmt.Errorf("%v: Quantity out of range: %v",
				errString, q.Quantity)
		}
	case FunctionReadInputRegisters:
		if q.Quantity == 0 || q.Quantity > 125 {
			return false, fmt.Errorf("%v: Quantity out of range: %v",
				errString, q.Quantity)
		}
	case FunctionWriteSingleCoil:
		if q.Quantity != 0 {
			return false, fmt.Errorf("%v: Quantity should be 0 but it is: %v",
				errString, q.Quantity)
		}
		if 4 != len(q.Data) {
			return false, fmt.Errorf("%v: len(Data) should be 4 but it is: %v",
				errString, len(q.Data))
		}
		if (0xFF != q.Data[2] && 0x00 != q.Data[2]) || 0x00 != q.Data[3] {
			return false, fmt.Errorf("%v: Data should be 0xFF00 or 0x0000 "+
				"but it is: 0x%x%x", errString, q.Data[0], q.Data[1])
		}
	case FunctionWriteSingleRegister:
		if q.Quantity != 0 {
			return false, fmt.Errorf("%v: Quantity should be 0 but it is: %v",
				errString, q.Quantity)
		}
		if 4 != len(q.Data) {
			return false, fmt.Errorf("%v: len(Data) should be 4 but it is: %v",
				errString, len(q.Data))
		}
	case FunctionWriteMultipleCoils:
		if q.Quantity == 0 || q.Quantity > 2000 {
			return false, fmt.Errorf("%v: Quantity out of range: %v",
				errString, q.Quantity)
		}
		expectedLen := 5 + q.Quantity/8
		if q.Quantity%8 != 0 {
			expectedLen++
		}
		if len(q.Data) != int(expectedLen) {
			return false, fmt.Errorf("%v: len(Data) should be %v but it is: "+
				"%v", errString, expectedLen, len(q.Data))
		}
	case FunctionWriteMultipleRegisters:
		if q.Quantity == 0 || q.Quantity > 126 {
			return false, fmt.Errorf("%v: Quantity out of range: %v",
				errString, q.Quantity)
		}
		expectedLen := int(5 + q.Quantity*2)
		if len(q.Data) != expectedLen {
			return false, fmt.Errorf("%v: len(Data) should be %v but it is: "+
				"%v", errString, expectedLen, len(q.Data))
		}
	case FunctionMaskWriteRegister:
	default:
		return false, fmt.Errorf("Invalid FunctionCode: %x", q.FunctionCode)
	}
	return true, nil
}

func (q *Query) IsRead() bool {
	switch q.FunctionCode {
	case FunctionReadCoils:
		fallthrough
	case FunctionReadDiscreteInputs:
		fallthrough
	case FunctionReadHoldingRegisters:
		fallthrough
	case FunctionReadInputRegisters:
		return true
	}
	return false
}

func (q *Query) IsWrite() bool {
	switch q.FunctionCode {
	case FunctionWriteSingleCoil:
		fallthrough
	case FunctionWriteSingleRegister:
		fallthrough
	case FunctionWriteMultipleCoils:
		fallthrough
	case FunctionWriteMultipleRegisters:
		fallthrough
	case FunctionMaskWriteRegister:
		return true
	}
	return false
}
