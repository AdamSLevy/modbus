package modbus

import (
	"strings"
	"testing"
)

type testQuery struct {
	Query
	isValid bool
	test    string
	Data    []byte
}

func init() {
	for i := range testQueries {
		testQueries[i].SlaveID = 1
	}
}

var testQueries = []testQuery{
	testQuery{isValid: false, test: "FunctionCode=0xff", Query: Query{
		FunctionCode: FunctionCode(0xff),
	}},

	// Read Coils
	testQuery{isValid: false, test: "Quantity=0", Query: Query{
		FunctionCode: FunctionReadCoils,
	}},
	testQuery{isValid: true, test: "Min Quantity=1", Query: Query{
		FunctionCode: FunctionReadCoils,
		Address:      1,
		Quantity:     1,
	}, Data: []byte{0, 1, 0, 1}},
	testQuery{isValid: true, test: "Max Quantity=2000", Query: Query{
		FunctionCode: FunctionReadCoils,
		Address:      0,
		Quantity:     2000,
	}, Data: []byte{0, 0, 0x07, 0xD0}},
	testQuery{isValid: false, test: "Max Exceeded Address=1 Quantity=2000", Query: Query{
		FunctionCode: FunctionReadCoils,
		Address:      1,
		Quantity:     2000,
	}},
	testQuery{isValid: false, test: "Max Exceeded Quantity=2001", Query: Query{
		FunctionCode: FunctionReadCoils,
		Quantity:     2001,
	}},

	// Read Discrete Inputs
	testQuery{isValid: false, test: "Quantity=0", Query: Query{
		FunctionCode: FunctionReadDiscreteInputs,
	}},
	testQuery{isValid: true, test: "Min Quantity=1", Query: Query{
		FunctionCode: FunctionReadDiscreteInputs,
		Address:      1,
		Quantity:     1,
	}, Data: []byte{0, 1, 0, 1}},
	testQuery{isValid: true, test: "Max Quantity=2000", Query: Query{
		FunctionCode: FunctionReadDiscreteInputs,
		Address:      0,
		Quantity:     2000,
	}, Data: []byte{0, 0, 0x07, 0xD0}},
	testQuery{isValid: false, test: "Max Exceeded Address=1 Quantity=2000", Query: Query{
		FunctionCode: FunctionReadDiscreteInputs,
		Address:      1,
		Quantity:     2000,
	}},
	testQuery{isValid: false, test: "Max Exceeded Quantity=2001", Query: Query{
		FunctionCode: FunctionReadDiscreteInputs,
		Quantity:     2001,
	}},

	// Read Holding Registers
	testQuery{isValid: false, test: "Quantity=0", Query: Query{
		FunctionCode: FunctionReadHoldingRegisters,
	}},
	testQuery{isValid: true, test: "Min Quantity=1", Query: Query{
		FunctionCode: FunctionReadHoldingRegisters,
		Address:      1,
		Quantity:     1,
	}, Data: []byte{0, 1, 0, 1}},
	testQuery{isValid: true, test: "Max Quantity=125", Query: Query{
		FunctionCode: FunctionReadHoldingRegisters,
		Address:      0,
		Quantity:     125,
	}, Data: []byte{0, 0, 0, 125}},
	testQuery{isValid: false, test: "Max Exceeded Address=1 Quantity=125", Query: Query{
		FunctionCode: FunctionReadHoldingRegisters,
		Address:      1,
		Quantity:     125,
	}},
	testQuery{isValid: false, test: "Max Exceeded Quantity=126", Query: Query{
		FunctionCode: FunctionReadHoldingRegisters,
		Quantity:     126,
	}},

	// Read Input Registers
	testQuery{isValid: false, test: "Quantity=0", Query: Query{
		FunctionCode: FunctionReadInputRegisters,
	}},
	testQuery{isValid: true, test: "Min Quantity=1", Query: Query{
		FunctionCode: FunctionReadInputRegisters,
		Address:      1,
		Quantity:     1,
	}, Data: []byte{0, 1, 0, 1}},
	testQuery{isValid: true, test: "Max Quantity=125", Query: Query{
		FunctionCode: FunctionReadInputRegisters,
		Address:      0,
		Quantity:     125,
	}, Data: []byte{0, 0, 0, 125}},
	testQuery{isValid: false, test: "Max Exceeded Address=1 Quantity=125", Query: Query{
		FunctionCode: FunctionReadInputRegisters,
		Address:      1,
		Quantity:     125,
	}},
	testQuery{isValid: false, test: "Max Exceeded Quantity=126", Query: Query{
		FunctionCode: FunctionReadInputRegisters,
		Quantity:     126,
	}},

	// Write Single Coil
	testQuery{isValid: false, test: "Values=nil", Query: Query{
		FunctionCode: FunctionWriteSingleCoil,
	}},
	testQuery{isValid: true, test: "Value[0]=0", Query: Query{
		FunctionCode: FunctionWriteSingleCoil,
		Address:      1,
		Values:       []uint16{0},
	}, Data: []byte{0, 1, 0, 0}},
	testQuery{isValid: true, test: "Value[0]=1", Query: Query{
		FunctionCode: FunctionWriteSingleCoil,
		Address:      1,
		Values:       []uint16{1},
	}, Data: []byte{0, 1, 0xff, 0}},
	testQuery{isValid: false, test: "len(Values)=2", Query: Query{
		FunctionCode: FunctionWriteSingleCoil,
		Values:       []uint16{0, 0},
	}},

	// Write Single Register
	testQuery{isValid: false, test: "Values=nil", Query: Query{
		FunctionCode: FunctionWriteSingleRegister,
	}},
	testQuery{isValid: true, test: "len(Values)=1", Query: Query{
		FunctionCode: FunctionWriteSingleRegister,
		Address:      1,
		Values:       []uint16{1},
	}, Data: []byte{0, 1, 0, 1}},
	testQuery{isValid: false, test: "len(Values)=2", Query: Query{
		FunctionCode: FunctionWriteSingleRegister,
		Values:       []uint16{0, 0},
	}},

	// Write Multiple Coils
	testQuery{isValid: false, test: "Quantity=0", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Values:       []uint16{0},
	}},
	testQuery{isValid: false, test: "Values=nil", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Quantity:     1,
	}},
	testQuery{isValid: false, test: "len(Values)=0", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Quantity:     1,
		Values:       []uint16{},
	}},
	testQuery{isValid: true, test: "Min Quantity=1", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Address:      1,
		Quantity:     1,
		Values:       []uint16{0x8000},
	}, Data: []byte{0, 1, 0, 1, 1, 0x80}},
	testQuery{isValid: true, test: "Quantity=16", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Address:      1,
		Quantity:     16,
		Values:       []uint16{0x8180},
	}, Data: []byte{0, 1, 0, 16, 2, 0x81, 0x80}},
	testQuery{isValid: false, test: "Quantity=17 len(values)=1", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Quantity:     17,
		Values:       []uint16{0},
	}},
	testQuery{isValid: true, test: "Quantity=17 len(Values)=2", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Address:      1,
		Quantity:     17,
		Values:       []uint16{0x8182, 0x8000},
	}, Data: []byte{0, 1, 0, 17, 3, 0x81, 0x82, 0x80}},
	testQuery{isValid: false, test: "Quantity=17 len(Values)=3", Query: Query{
		FunctionCode: FunctionWriteMultipleCoils,
		Quantity:     17,
		Values:       []uint16{0, 0, 0},
	}},

	// Write Multiple Registers
	testQuery{isValid: false, test: "Quantity=0", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Values:       []uint16{0},
	}},
	testQuery{isValid: false, test: "Values=nil", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Quantity:     1,
	}},
	testQuery{isValid: false, test: "len(Values)=0", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Quantity:     1,
		Values:       []uint16{},
	}},
	testQuery{isValid: true, test: "Min Quantity=1", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Address:      1,
		Quantity:     1,
		Values:       []uint16{0x8081},
	}, Data: []byte{0, 1, 0, 1, 2, 0x80, 0x81}},
	testQuery{isValid: false, test: "Quantity=2 len(values)=1", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Quantity:     2,
		Values:       []uint16{0},
	}},
	testQuery{isValid: true, test: "Quantity=2 len(Values)=2", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Address:      1,
		Quantity:     2,
		Values:       []uint16{0x8081, 0x7071},
	}, Data: []byte{0, 1, 0, 2, 4, 0x80, 0x81, 0x70, 0x71}},
	testQuery{isValid: false, test: "Quantity=2 len(Values)=3", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Quantity:     2,
		Values:       []uint16{0, 0, 0},
	}},
	testQuery{isValid: true, test: "Max Quantity=123", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Address:      0,
		Quantity:     123,
		Values:       make([]uint16, 123),
	}, Data: []byte{0, 0, 0, 123, 246,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,

		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,

		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0,
	}},
	testQuery{isValid: false, test: "Max Exceeded Address=1 Quantity=123", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Address:      1,
		Quantity:     123,
		Values:       make([]uint16, 123),
	}},
	testQuery{isValid: false, test: "Max Exceeded Quantity=124", Query: Query{
		FunctionCode: FunctionWriteMultipleRegisters,
		Quantity:     124,
		Values:       make([]uint16, 124),
	}},

	// Mask Write Register
	testQuery{isValid: false, test: "Values=nil", Query: Query{
		FunctionCode: FunctionMaskWriteRegister,
	}},
	testQuery{isValid: false, test: "len(Values)=0", Query: Query{
		FunctionCode: FunctionMaskWriteRegister,
		Values:       []uint16{},
	}},
	testQuery{isValid: false, test: "len(Values)=1", Query: Query{
		FunctionCode: FunctionMaskWriteRegister,
		Values:       []uint16{0},
	}},
	testQuery{isValid: true, test: "len(Values)=2", Query: Query{
		FunctionCode: FunctionMaskWriteRegister,
		Address:      1,
		Values:       []uint16{0x1112, 0x2122},
	}, Data: []byte{0, 1, 0x11, 0x12, 0x21, 0x22}},
	testQuery{isValid: false, test: "len(Values)=3", Query: Query{
		FunctionCode: FunctionMaskWriteRegister,
		Values:       []uint16{0, 0, 0},
	}},
}

func TestQuery(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		for _, q := range testQueries {
			fName, ok := FunctionNames[q.FunctionCode]
			if !ok {
				fName = "Invalid"
			}
			testName := fName + "/" + q.test
			t.Run(testName, func(t *testing.T) {
				testIsValid(t, q)
			})
		}
	})
	t.Run("data", func(t *testing.T) {
		for _, q := range testQueries {
			fName, ok := FunctionNames[q.FunctionCode]
			if !ok {
				fName = "Invalid"
			}
			testName := fName + "/" + q.test
			t.Run(testName, func(t *testing.T) {
				testData(t, q)
			})
		}
	})
	t.Run("isValidResponse", func(t *testing.T) {
		tested := make(map[FunctionCode]bool)
		for _, q := range testQueries {
			if q.isValid && !tested[q.FunctionCode] {
				tested[q.FunctionCode] = true
				fName, ok := FunctionNames[q.FunctionCode]
				if !ok {
					fName = "Invalid"
				}
				t.Run(fName, func(t *testing.T) {
					testResponses(t, q)
				})
			}
		}
	})
	t.Run("WriteSingleCoil", func(t *testing.T) {
		t.Run("false", func(t *testing.T) { testWriteSingleCoil(t, false) })
		t.Run("true", func(t *testing.T) { testWriteSingleCoil(t, true) })
	})
	t.Run("WriteSingleQuery", func(t *testing.T) {
		for _, fCode := range FunctionCodes {
			fCode := fCode
			t.Run(FunctionNames[fCode], func(t *testing.T) {
				_, err := WriteSingleQuery(0, fCode, 0, 0)
				switch fCode {
				case FunctionWriteSingleCoil:
					fallthrough
				case FunctionWriteSingleRegister:
					if nil != err {
						t.Errorf("err is nil")
					}
				default:
					if nil == err {
						t.Errorf("err is nil")
					}
				}
			})
		}
	})
	t.Run("WriteMultipleQuery", func(t *testing.T) {
		for _, fCode := range FunctionCodes {
			fCode := fCode
			t.Run(FunctionNames[fCode], func(t *testing.T) {
				_, err := WriteMultipleQuery(0, fCode, 0, 1,
					[]uint16{0})
				switch fCode {
				case FunctionWriteMultipleCoils:
					fallthrough
				case FunctionWriteMultipleRegisters:
					if nil != err {
						t.Errorf("err is nil")
					}
				default:
					if nil == err {
						t.Errorf("err is nil")
					}
				}
			})
		}
	})
	t.Run("ReadQuery", func(t *testing.T) {
		for _, fCode := range FunctionCodes {
			fCode := fCode
			t.Run(FunctionNames[fCode], func(t *testing.T) {
				_, err := ReadQuery(0, fCode, 0, 1)
				switch fCode {
				case FunctionReadCoils:
					fallthrough
				case FunctionReadDiscreteInputs:
					fallthrough
				case FunctionReadInputRegisters:
					fallthrough
				case FunctionReadHoldingRegisters:
					if nil != err {
						t.Errorf("err is nil")
					}
				default:
					if nil == err {
						t.Errorf("err is nil")
					}
				}
			})
		}
	})
	t.Run("ReadCoils", func(t *testing.T) {
		if _, err := ReadCoils(0, 0, 1); nil != err {
			t.Error(err)
		}
	})
	t.Run("ReadDiscreteInputs", func(t *testing.T) {
		if _, err := ReadDiscreteInputs(0, 0, 1); nil != err {
			t.Error(err)
		}
	})
	t.Run("ReadHoldingRegisters", func(t *testing.T) {
		if _, err := ReadHoldingRegisters(0, 0, 1); nil != err {
			t.Error(err)
		}
	})
	t.Run("ReadInputRegisters", func(t *testing.T) {
		if _, err := ReadInputRegisters(0, 0, 1); nil != err {
			t.Error(err)
		}
	})
	t.Run("WriteSingleRegister", func(t *testing.T) {
		if _, err := WriteSingleRegister(0, 0, 1); nil != err {
			t.Error(err)
		}
	})
	t.Run("WriteMultipleRegisters", func(t *testing.T) {
		if _, err := WriteMultipleRegisters(0, 0, 1, []uint16{0}); nil != err {
			t.Error(err)
		}
	})
	t.Run("WriteMultipleCoils", func(t *testing.T) {
		if _, err := WriteMultipleCoils(0, 0, 1, []uint16{0}); nil != err {
			t.Error(err)
		}
	})
	t.Run("MaskWriteRegister", func(t *testing.T) {
		if _, err := MaskWriteRegister(0, 0, 1, 0); nil != err {
			t.Error(err)
		}
	})
}

func testIsValid(t *testing.T, q testQuery) {
	valid, err := q.IsValid()
	if q.isValid {
		if !valid {
			t.Errorf("Query%v bool want: %v, got: %v",
				q.Query, q.isValid, valid)
		}
		if nil != err {
			t.Errorf("Query%v error want: %v got: %v",
				q.Query, nil, err)
		}
	} else {
		if valid {
			t.Errorf("Query%v bool: want: %v, got: %v",
				q.Query, q.isValid, valid)
		}
		if nil == err {
			t.Errorf("Query%v error is nil", q.Query)
		}
	}
}

func testResponses(t *testing.T, q testQuery) {
	if !q.isValid {
		t.Fatal("testIsValidResponse requires a valid Query")
	}
	for i, e := range exceptions {
		i := i
		e := e
		switch i {
		case exceptionUnknown:
			response := []byte{
				q.SlaveID,
				byte(q.FunctionCode) + 0x95,
				0xaa,
			}
			t.Run(e.Error()+"/Bad Function Code", func(t *testing.T) {
				testIsValidResponse(t, q.Query, response, e)
			})
			response = []byte{
				q.SlaveID,
				byte(q.FunctionCode) + 0x80,
				0xaa,
			}
			t.Run(e.Error(), func(t *testing.T) {
				testIsValidResponse(t, q.Query, response, e)
			})
		case exceptionEmptyResponse:
			response := []byte{}
			t.Run(e.Error()+"/len(response)=0", func(t *testing.T) {
				testIsValidResponse(t, q.Query, response, e)
			})
			t.Run(e.Error()+"/response=nil", func(t *testing.T) {
				testIsValidResponse(t, q.Query, nil, e)
			})
		case exceptionBadResponseLength:
			if IsReadFunction(q.FunctionCode) {
				response := []byte{
					q.SlaveID,
					byte(q.FunctionCode),
				}
				var response1, response2 []byte
				switch q.FunctionCode {
				case FunctionReadCoils:
					fallthrough
				case FunctionReadDiscreteInputs:
					response1 = append(response, 0)
					response2 = append(response, 2, 1, 1)
				case FunctionReadInputRegisters:
					fallthrough
				case FunctionReadHoldingRegisters:
					response1 = append(response, 1, 0)
					response2 = append(response, 4, 0, 1, 0, 1)
				}
				t.Run(e.Error()+"/Too Short", func(t *testing.T) {
					testIsValidResponse(t, q.Query, response1, e)
				})
				t.Run(e.Error()+"/Too Long", func(t *testing.T) {
					testIsValidResponse(t, q.Query, response2, e)
				})
			}
		case exceptionResponseLengthMismatch:
			if IsReadFunction(q.FunctionCode) {
				response := []byte{
					q.SlaveID,
					byte(q.FunctionCode),
				}
				var response1, response2 []byte
				switch q.FunctionCode {
				case FunctionReadCoils:
					fallthrough
				case FunctionReadDiscreteInputs:
					response1 = append(response, 1)
					response2 = append(response, 1, 1, 1)
				case FunctionReadInputRegisters:
					fallthrough
				case FunctionReadHoldingRegisters:
					response1 = append(response, 2, 0)
					response2 = append(response, 2, 0, 1, 0)
				}
				t.Run(e.Error()+"/Too Short", func(t *testing.T) {
					testIsValidResponse(t, q.Query, response1, e)
				})
				t.Run(e.Error()+"/Too Long", func(t *testing.T) {
					testIsValidResponse(t, q.Query, response2, e)
				})
			}
		case exceptionSlaveIDMismatch:
			response := []byte{
				q.SlaveID + 1,
				byte(q.FunctionCode) + 0x80,
				byte(i),
			}
			t.Run(e.Error(), func(t *testing.T) {
				testIsValidResponse(t, q.Query, response, e)
			})
		case exceptionWriteDataMismatch:
			if IsWriteFunction(q.FunctionCode) {
				data, _ := q.data()
				data[2] = 0xfe
				response := append([]byte{
					q.SlaveID,
					byte(q.FunctionCode),
				}, data...)
				t.Run(e.Error(), func(t *testing.T) {
					testIsValidResponse(t, q.Query, response, e)
				})
			}
		case exceptionBadFraming:
			// Not returned from isValidResponse
		case exceptionBadChecksum:
			// Not returned from isValidResponse
		default:
			response := []byte{
				q.SlaveID,
				byte(q.FunctionCode) + 0x80,
				byte(i),
			}
			t.Run(e.Error(), func(t *testing.T) {
				testIsValidResponse(t, q.Query, response, e)
			})
		}
	}
}

func testIsValidResponse(t *testing.T, q Query, response []byte, e error) {
	valid, err := q.isValidResponse(response)
	if valid {
		t.Error("Modbus Error Response marked valid")
	}
	if nil == err {
		t.Error("err = nil")
	} else if 0 != strings.Compare(e.Error(), err.Error()) {
		t.Error("Exception mismatch:", err)
	}
}

func testData(t *testing.T, q testQuery) {
	data, err := q.data()
	if q.isValid {
		if nil != err {
			t.Errorf("Query%v error want: %v got: %v",
				q.Query, nil, err)
		}
		if nil == data {
			t.Fatalf("Query%v data is nil", q.Query)
		}
		for i := range data {
			if i == len(q.Data) ||
				data[i] != q.Data[i] {
				t.Fatalf("Query%v data want: %v, got: %v",
					q.Query, q.Data, data)
			}
		}
	} else {
		if nil != data {
			t.Errorf("Query%v data is not nil", q.Query)
		}
		if nil == err {
			t.Errorf("Query%v error is nil", q.Query)
		}
	}
}

func testWriteSingleCoil(t *testing.T, coil bool) {
	q, err := WriteSingleCoil(0, 0, coil)
	if nil != err {
		t.Fatal(err)
	}
	if len(q.Values) != 1 {
		if nil == q.Values {
			t.Error("Values is nil")
		} else {
			t.Errorf("len(Values) want: 1 got: %v", len(q.Values))
		}
	} else {
		if coil {
			if 0xFF00 != q.Values[0] {
				t.Error("Values[0] != 0xFF00")
			}
		} else {
			if 0 != q.Values[0] {
				t.Error("Values[0] != 0")
			}
		}
	}
}
