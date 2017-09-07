package modbus

import (
	"errors"
	"fmt"
)

// Query contains the raw information for a Modbus query and the Response
// channel to receive the response data on.
type Query struct {
	SlaveID      byte
	FunctionCode byte
	Address      uint16
	Quantity     uint16
	Data         []byte
	Response     chan *QueryResponse
}

// NewQuery returns a pointer to an initialized Query with a valid Response
// channel for receiving the QueryResponse.
func NewQuery() *Query {
	return &Query{
		Response: make(chan *QueryResponse),
	}
}

// sendResponse is a convenience function for sending a ConnectionResponse.
func (q *Query) sendResponse(res *QueryResponse) {
	q.Response <- res
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
	switch q.FunctionCode {
	case FunctionReadCoils:
		if q.Quantity == 0 || q.Quantity > 2000 {
			return false, fmt.Errorf("Quantity out of range: %v", q.Quantity)
		}
	case FunctionReadDiscreteInputs:
		if q.Quantity == 0 || q.Quantity > 2000 {
			return false, fmt.Errorf("Quantity out of range: %v", q.Quantity)
		}
	case FunctionReadHoldingRegisters:
		if q.Quantity == 0 || q.Quantity > 126 {
			return false, fmt.Errorf("Quantity out of range: %v", q.Quantity)
		}
	case FunctionReadInputRegisters:
		if q.Quantity == 0 || q.Quantity > 126 {
			return false, fmt.Errorf("Quantity out of range: %v", q.Quantity)
		}
	case FunctionWriteSingleCoil:
		if q.Quantity != 0 {
			return false, fmt.Errorf("Quantity should be 0 but it is: %v",
				q.Quantity)
		}
		if 2 != len(q.Data) {
			return false, fmt.Errorf("len(Data) should be 2 but it is: %v",
				len(q.Data))
		}
		if (0xFF != q.Data[0] && 0x00 != q.Data[0]) || 0x00 != q.Data[1] {
			return false, fmt.Errorf("Data should be 0xFF00 or 0x0000 but "+
				"it is: 0x%x%x", q.Data[0], q.Data[1])
		}
	case FunctionWriteSingleRegister:
		if q.Quantity != 0 {
			return false, fmt.Errorf("Quantity should be 0 but it is: %v",
				q.Quantity)
		}
		if 2 != len(q.Data) {
			return false, fmt.Errorf("len(Data) should be 2 but it is: %v",
				len(q.Data))
		}
	case FunctionWriteMultipleSingleCoils:
		if q.Quantity == 0 || q.Quantity > 2000 {
			return false, fmt.Errorf("Quantity out of range: %v", q.Quantity)
		}
		expectedLen := q.Quantity / 8
		if q.Quantity%8 != 0 {
			expectedLen++
		}
		if len(q.Data) != int(expectedLen) {
			return false, fmt.Errorf("len(Data) should be %v but it is: %v",
				expectedLen, len(q.Data))
		}
	case FunctionWriteMultipleRegisters:
		if q.Quantity == 0 || q.Quantity > 126 {
			return false, fmt.Errorf("Quantity out of range: %v", q.Quantity)
		}
		if len(q.Data) != int(2*q.Quantity) {
			return false, fmt.Errorf("len(Data) should be %v but it is: %v",
				2*q.Quantity, len(q.Data))
		}
	default:
		return false, fmt.Errorf("Invalid FunctionCode: %x", q.FunctionCode)
	}
	return true, nil
}

// ValidReadFunction returns a boolean, depending on whether or not the
// given code corresponds to a valid modbus read function code
func (q *Query) ValidReadFunction() (bool, error) {
	if q.FunctionCode < FunctionReadCoils ||
		q.FunctionCode > FunctionReadInputRegisters {
		return false, errors.New("Not a valid read function")
	}

	return q.IsValid()
}

// ValidWriteFunction returns a boolean, depending on whether or not the
// given code corresponds to a valid modbus write function code
func (q *Query) ValidWriteFunction() (bool, error) {
	if q.FunctionCode < FunctionWriteSingleCoil ||
		q.FunctionCode > FunctionWriteMultipleRegisters {
		return false, errors.New("Not a valid write function")
	}

	return q.IsValid()
}
