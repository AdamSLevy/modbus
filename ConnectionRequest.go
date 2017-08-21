package modbus

// ConnectionRequests are sent to ClientManager's runtime to get a
// ConnectionResponse back on the Response channel containing a QueryQueue that
// can be used to send asynchronous Modbus Queries to a Client with the
// specified Connection. Upon receiving a ConnectionRequest the ClientManager
// checks if a Client with the same Connection.Host name already exists. If an
// existing Client is found, all other Connection settings must match for a
// successful ClientResponse. If no such Client is found, a new Client is
// created if the Connection is set up successfully. A ConnectionResponse will
// always be returned to the caller on the Response channel with either a
// QueryQueue channel or an error.
type ConnectionRequest struct {
	Connection
	Response chan *ConnectionResponse
}

// NewConnectionRequest creates a new ConnectionRequest with an initialized
// Response channel. User must then set the Client settings directly.
func NewConnectionRequest() *ConnectionRequest {
	return &ConnectionRequest{
		Response: make(chan *ConnectionResponse),
	}
}

// sendResponse is a convenience function for sending a ClientResponse.
func (req *ConnectionRequest) sendResponse(q chan *Query, err error) {
	req.Response <- &ConnectionResponse{
		QueryQueue: q,
		Err:        err,
	}
}

// ConnectionResponse is returned on the Response channel of a previously sent
// ConnectionRequest. On success, Err is nil and the QueryQueue channel can be
// used to send Queries to a Client with the requested Connection.
type ConnectionResponse struct {
	QueryQueue chan *Query
	Err        error
}
