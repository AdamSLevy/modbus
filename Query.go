package modbus

import (
	"encoding/binary"
	"fmt"
)

// Query contains the necessary data for a Packager to construct and execute a
// Modbus query.
type Query struct {
	FunctionCode
	SlaveID  byte
	Address  uint16
	Quantity uint16
	Values   []uint16
}

// IsValid returns a bool representing whether the Query is well constructed
// with a supported FunctionCode and appropriate Quantity and len(Values). If
// the query is invalid, IsValid returns false, and an error describing the
// reason for not passing. Otherwise it returns true, nil.
func (q Query) IsValid() (bool, error) {
	errString, _ := FunctionNames[q.FunctionCode]
	var maxQuantity uint16
	var expectedLen int
	switch q.FunctionCode {
	case FunctionReadCoils:
		fallthrough
	case FunctionReadDiscreteInputs:
		maxQuantity = 2000
	case FunctionReadHoldingRegisters:
		fallthrough
	case FunctionReadInputRegisters:
		maxQuantity = 125
	case FunctionWriteSingleCoil:
		expectedLen = 1
	case FunctionWriteSingleRegister:
		expectedLen = 1
	case FunctionWriteMultipleCoils:
		maxQuantity = 2000
		expectedLen = int(q.Quantity) / 16
		if q.Quantity%16 != 0 {
			expectedLen++
		}
	case FunctionWriteMultipleRegisters:
		maxQuantity = 125
		expectedLen = int(q.Quantity)
	case FunctionMaskWriteRegister:
		expectedLen = 2
	default:
		return false, fmt.Errorf("Invalid FunctionCode: %x", q.FunctionCode)
	}

	// Check quantity
	if IsWriteMultipleFunction(q.FunctionCode) &&
		(q.Quantity == 0 || q.Quantity > maxQuantity) {
		return false, fmt.Errorf("%v: Quantity %v out of range [1, %v]",
			errString, q.Quantity, maxQuantity)
	}
	// Check len(Values)
	if IsWriteFunction(q.FunctionCode) {
		if len(q.Values) != expectedLen {
			return false, fmt.Errorf(
				"%v: len(Values) should be %v but it is: %v",
				errString, expectedLen, len(q.Values))
		}
	}

	return true, nil
}

// data s called by a Packager to construct the data payload for the Query and
// check if it IsValid().
func (q Query) data() ([]byte, error) {
	if valid, err := q.IsValid(); !valid {
		return nil, err
	}
	if IsReadFunction(q.FunctionCode) {
		return dataBlock(q.Address, q.Quantity), nil
	}
	if IsWriteFunction(q.FunctionCode) {
		if IsWriteMultipleFunction(q.FunctionCode) {
			values := dataBlock(q.Values...)
			if FunctionWriteMultipleCoils == q.FunctionCode {
				numValues := q.Quantity / 8
				if q.Quantity%8 != 0 {
					numValues++
				}
				values = values[0:numValues]
			}
			return dataBlockSuffix(values, q.Address, q.Quantity), nil
		}
		if IsWriteSingleFunction(q.FunctionCode) {
			if q.FunctionCode == FunctionWriteSingleCoil && q.Values[0] != 0 {
				q.Values[0] = 0xFF00
			}
			return dataBlock(q.Address, q.Values[0]), nil
		}
		if q.FunctionCode == FunctionMaskWriteRegister {
			andMask := q.Values[0]
			orMask := q.Values[1]
			return dataBlock(q.Address, andMask, orMask), nil
		}
	}
	return nil, fmt.Errorf("Invalid FunctionCode: %x", q.FunctionCode)
}

// dataBlock creates a sequence of uint16 data.
func dataBlock(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// dataBlockSuffix creates a sequence of uint16 data and appends 1 byte for
// len(suffix) and then the suffix.
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

// ReadQuery constructs a Query where IsReadFunction(fCode) is true.
func ReadQuery(slaveID byte, fCode FunctionCode,
	address, quantity uint16) (Query, error) {
	q := Query{
		SlaveID:      slaveID,
		FunctionCode: fCode,
		Address:      address,
		Quantity:     quantity,
	}
	if !IsReadFunction(fCode) {
		return q, fmt.Errorf("Not a valid read function code")
	}
	_, err := q.IsValid()
	return q, err
}

// WriteSingleQuery constructs a Query where IsWriteSingleFunction(fCode) is
// true.
func WriteSingleQuery(slaveID byte, fCode FunctionCode,
	address, value uint16) (Query, error) {
	q := Query{
		SlaveID:      slaveID,
		FunctionCode: fCode,
		Address:      address,
		Values:       []uint16{value},
	}
	if !IsWriteSingleFunction(fCode) {
		return q, fmt.Errorf("Not a single write function code")
	}
	_, err := q.IsValid()
	return q, err
}

// WriteMultipleQuery constructs a Query where IsWriteMultipleFunction(fCode)
// is true.
func WriteMultipleQuery(slaveID byte, fCode FunctionCode,
	address, quantity uint16, values []uint16) (Query, error) {
	q := Query{
		SlaveID:      slaveID,
		FunctionCode: fCode,
		Address:      address,
		Quantity:     quantity,
		Values:       values,
	}
	if !IsWriteMultipleFunction(fCode) {
		return q, fmt.Errorf("Not a multiple write function code")
	}
	_, err := q.IsValid()
	return q, err
}

// ReadCoils constructs a ReadCoils Query object.
func ReadCoils(slaveID byte, address, quantity uint16) (Query, error) {
	return ReadQuery(slaveID, FunctionReadCoils, address, quantity)
}

// ReadDiscreteInputs constructs a ReadDiscreteInputs Query object.
func ReadDiscreteInputs(slaveID byte, address, quantity uint16) (Query, error) {
	return ReadQuery(slaveID, FunctionReadDiscreteInputs, address, quantity)
}

// ReadHoldingRegisters constructs a ReadHoldingRegisters Query object.
func ReadHoldingRegisters(slaveID byte, address, quantity uint16) (Query, error) {
	return ReadQuery(slaveID, FunctionReadHoldingRegisters, address, quantity)
}

// ReadInputRegisters constructs a ReadInputRegisters Query object.
func ReadInputRegisters(slaveID byte, address, quantity uint16) (Query, error) {
	return ReadQuery(slaveID, FunctionReadInputRegisters, address, quantity)
}

// WriteSingleCoil constructs a WriteSingleCoil Query object.
func WriteSingleCoil(slaveID byte, address uint16, value bool) (Query, error) {
	var valueData uint16
	if value {
		valueData = 0xFF00
	}
	return WriteSingleQuery(slaveID, FunctionWriteSingleCoil, address, valueData)
}

// WriteSingleRegister constructs a WriteSingleRegister Query object.
func WriteSingleRegister(slaveID byte, address, value uint16) (Query, error) {
	return WriteSingleQuery(slaveID, FunctionWriteSingleRegister, address, value)
}

// WriteMultipleCoils constructs a WriteMultipleCoils Query object.
func WriteMultipleCoils(slaveID byte, address, quantity uint16,
	value []uint16) (Query, error) {
	return WriteMultipleQuery(slaveID, FunctionWriteMultipleCoils,
		address, quantity, value)
}

// WriteMultipleRegisters constructs a WriteMultipleRegisters Query object.
func WriteMultipleRegisters(slaveID byte, address, quantity uint16,
	value []uint16) (Query, error) {
	return WriteMultipleQuery(slaveID, FunctionWriteMultipleRegisters,
		address, quantity, value)
}

// MaskWriteRegister constructs a MaskWriteRegister Query object.
func MaskWriteRegister(slaveID byte, address, andMask, orMask uint16) (Query, error) {
	q := Query{
		SlaveID:      slaveID,
		FunctionCode: FunctionMaskWriteRegister,
		Address:      address,
		Values:       []uint16{andMask, orMask},
	}
	_, err := q.IsValid()
	return q, err
}

// IsReadFunction returns true if fCode is FunctionReadCoils,
// FunctionReadDiscreteInputs, FunctionReadHoldingRegisters, or
// FunctionReadInputRegisters.
func IsReadFunction(fCode FunctionCode) bool {
	switch fCode {
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

// IsWriteFunction returns true if fCode is FunctionWriteSingleCoil,
// FunctionWriteSingleRegister, FunctionWriteMultipleCoils,
// FunctionWriteMultipleRegisters, or FunctionMaskWriteRegister.
func IsWriteFunction(fCode FunctionCode) bool {
	switch fCode {
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

// IsWriteSingleFunction returns true if fCode is FunctionWriteSingleCoil or
// FunctionWriteSingleRegister
func IsWriteSingleFunction(fCode FunctionCode) bool {
	switch fCode {
	case FunctionWriteSingleCoil:
		fallthrough
	case FunctionWriteSingleRegister:
		return true
	}
	return false
}

// IsWriteMultipleFunction returns true if fCode is FunctionWriteMultipleCoils
// or FunctionWriteMultipleRegisters.
func IsWriteMultipleFunction(fCode FunctionCode) bool {
	switch fCode {
	case FunctionWriteMultipleCoils:
		fallthrough
	case FunctionWriteMultipleRegisters:
		return true
	}
	return false
}
