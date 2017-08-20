package modbus

// Query contains the raw information for a Modbus query and the Response
// channel to receive the response data on.
type Query struct {
	SlaveId      SlaveId_t
	FunctionCode uint8
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
