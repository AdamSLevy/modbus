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
		maxQuantity = 123
		expectedLen = int(q.Quantity)
	case FunctionMaskWriteRegister:
		expectedLen = 2
	default:
		return false, fmt.Errorf("Invalid FunctionCode: %x", q.FunctionCode)
	}

	// Check quantity
	if (isReadFunction(q.FunctionCode) || isWriteMultipleFunction(q.FunctionCode)) &&
		(q.Quantity == 0 || q.Address+q.Quantity > maxQuantity) {
		return false, fmt.Errorf("%v: Requested address range [%v, %v] out of range [0, %v]",
			errString, q.Address, q.Address+q.Quantity, maxQuantity)
	}
	// Check len(Values)
	if isWriteFunction(q.FunctionCode) {
		if len(q.Values) != expectedLen {
			return false, fmt.Errorf(
				"%v: len(Values) should be %v but it is: %v",
				errString, expectedLen, len(q.Values))
		}
	}

	return true, nil
}

// isValidResponse is used by Packagers to validate the response data against a
// given Query.
func (q Query) isValidResponse(response []byte) (bool, error) {
	if nil == response || len(response) == 0 {
		return false, exceptions[exceptionEmptyResponse]
	}

	if response[0] != q.SlaveID {
		return false, exceptions[exceptionSlaveIDMismatch]
	}

	// Check for Modbus Exception Response
	if FunctionCode(response[1]) != q.FunctionCode {
		if FunctionCode(response[1]&0x7f) == q.FunctionCode {
			switch response[2] {
			case exceptionIllegalFunction:
				fallthrough
			case exceptionDataAddress:
				fallthrough
			case exceptionDataValue:
				fallthrough
			case exceptionSlaveDeviceFailure:
				fallthrough
			case exceptionAcknowledge:
				fallthrough
			case exceptionSlaveDeviceBusy:
				fallthrough
			case exceptionMemoryParityError:
				fallthrough
			case exceptionGatewayPathUnavailable:
				fallthrough
			case exceptionGatewayTargetDeviceFailedToRespond:
				return false, exceptions[uint16(response[2])]
			default:
				return false, exceptions[exceptionUnknown]
			}
		}
		return false, exceptions[exceptionUnknown]
	}

	if isWriteFunction(q.FunctionCode) {
		data, _ := q.data()
		for i := 0; i < 4; i++ {
			if i >= len(response) || data[i] != response[2+i] {
				return false, exceptions[exceptionWriteDataMismatch]
			}
		}
	}

	if isReadFunction(q.FunctionCode) {
		var expectedLen int
		switch q.FunctionCode {
		case FunctionReadCoils:
			fallthrough
		case FunctionReadDiscreteInputs:
			expectedLen = int(q.Quantity) / 8
			if q.Quantity%8 != 0 {
				expectedLen++
			}
		case FunctionReadInputRegisters:
			fallthrough
		case FunctionReadHoldingRegisters:
			expectedLen = int(q.Quantity) * 2
		}
		if int(response[2]) != expectedLen {
			return false, exceptions[exceptionBadResponseLength]
		}
		if len(response[3:]) != expectedLen {
			return false, exceptions[exceptionResponseLengthMismatch]
		}
	}

	return true, nil
}

// data is called by a Packager to construct the data payload for the Query and
// check if it IsValid().
func (q Query) data() ([]byte, error) {
	if valid, err := q.IsValid(); !valid {
		return nil, err
	}
	if isWriteFunction(q.FunctionCode) {
		if isWriteMultipleFunction(q.FunctionCode) {
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
		if isWriteSingleFunction(q.FunctionCode) {
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

	// isReadFunction() must be true
	return dataBlock(q.Address, q.Quantity), nil
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

// ReadQuery constructs a Query where isReadFunction(fCode) is true.
func ReadQuery(slaveID byte, fCode FunctionCode,
	address, quantity uint16) (Query, error) {
	q := Query{
		SlaveID:      slaveID,
		FunctionCode: fCode,
		Address:      address,
		Quantity:     quantity,
	}
	if !isReadFunction(fCode) {
		return q, fmt.Errorf("Not a valid read function code")
	}
	_, err := q.IsValid()
	return q, err
}

// WriteSingleQuery constructs a Query where isWriteSingleFunction(fCode) is
// true.
func WriteSingleQuery(slaveID byte, fCode FunctionCode,
	address, value uint16) (Query, error) {
	q := Query{
		SlaveID:      slaveID,
		FunctionCode: fCode,
		Address:      address,
		Values:       []uint16{value},
	}
	if !isWriteSingleFunction(fCode) {
		return q, fmt.Errorf("Not a single write function code")
	}
	_, err := q.IsValid()
	return q, err
}

// WriteMultipleQuery constructs a Query where isWriteMultipleFunction(fCode)
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
	if !isWriteMultipleFunction(fCode) {
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

// isReadFunction returns true if fCode is FunctionReadCoils,
// FunctionReadDiscreteInputs, FunctionReadHoldingRegisters, or
// FunctionReadInputRegisters.
func isReadFunction(fCode FunctionCode) bool {
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

// isWriteFunction returns true if fCode is FunctionWriteSingleCoil,
// FunctionWriteSingleRegister, FunctionWriteMultipleCoils,
// FunctionWriteMultipleRegisters, or FunctionMaskWriteRegister.
func isWriteFunction(fCode FunctionCode) bool {
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

// isWriteSingleFunction returns true if fCode is FunctionWriteSingleCoil or
// FunctionWriteSingleRegister
func isWriteSingleFunction(fCode FunctionCode) bool {
	switch fCode {
	case FunctionWriteSingleCoil:
		fallthrough
	case FunctionWriteSingleRegister:
		return true
	}
	return false
}

// isWriteMultipleFunction returns true if fCode is FunctionWriteMultipleCoils
// or FunctionWriteMultipleRegisters.
func isWriteMultipleFunction(fCode FunctionCode) bool {
	switch fCode {
	case FunctionWriteMultipleCoils:
		fallthrough
	case FunctionWriteMultipleRegisters:
		return true
	}
	return false
}
