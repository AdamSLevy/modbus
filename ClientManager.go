package modbus

import (
	"errors"
	"fmt"
	"sync"
)

// ClientManager is a singleton object that keeps track of Clients
// globally for the entire program. Use GetClientManager() to get a pointer
// to the global ClientManager singleton. Clients are hashed on their
// Host field. Clients are set up and accessed by sending
// ClientRequests to the ClientManager goroutine using SendRequest().
type ClientManager struct {
	newClient chan *ClientRequest
	clients   map[string]Client
}

var clientManager *ClientManager
var once sync.Once

// GetClientManager returns a pointer to the singleton instance of
// ClientManager, initializing and starting the ClientManager goroutine
// if necessary.
func GetClientManager() *ClientManager {
	once.Do(func() {
		clientManager = &ClientManager{
			newClient: make(chan *ClientRequest),
			clients:   make(map[string]Client),
		}

		go clientManager.requestListener()
	})

	return clientManager
}

// SendRequest sends a ClientRequest to the ClientManager runtime. The
// caller should expect a ClientResponse on the Response channel.
func (cm *ClientManager) SendRequest(req *ClientRequest) error {
	if nil == cm.newClient {
		return errors.New("Uninitialized ClientManager")
	}
	go func() {
		cm.newClient <- req
	}()
	return nil
}

// ClientRequests are sent to ClientManager's runtime to get a
// ClientResponse back on the Response channel. The Client is set up if
// it does not exist. If a Client with the same Host already exists, all
// settings must match for a successful ClientResponse.
type ClientRequest struct {
	Client
	Response chan *ClientResponse
}

// NewClientRequest creates a new ClientRequest with an initialized
// Response channel. User must then set the Client settings directly.
func NewClientRequest() *ClientRequest {
	return &ClientRequest{
		Response: make(chan *ClientResponse),
	}
}

// sendResponse is a convenience function for sending a ClientResponse.
func (req *ClientRequest) sendResponse(res *ClientResponse) {
	req.Response <- res
}

// ClientResponse contains the QueryQueue channel for the Client
// requested in a ClientRequest previously sent to the ClientManager.
// The QueryQueue channel can then be used to queue a Query on the Client
// resource.
type ClientResponse struct {
	QueryQueue chan *Query
	Err        error
}

// requestListener listens for incoming ClientRequests and sends a
// ClientResponse to the ClientRequest.Response channel. On success,
// the ClientResponse has a valid QueryQueue channel for sending queries to
// the requested client. On failure, ClientResponse.Error is set.
// Failure will occur if the client fails or if a client for the
// requested Host already exists with different settings. Existing clients
// can only be requested if all settings match exactly.
func (cm *ClientManager) requestListener() {
	for conReq := range cm.newClient {
		if nil == conReq.Response {
			continue
		}
		con, ok := cm.clients[conReq.Host]
		if ok {
			if con.Mode == conReq.Mode &&
				con.Baud == conReq.Baud {
				conReq.Response <- &ClientResponse{
					QueryQueue: con.queryQueue,
				}
			} else {
				// Host is in use but other
				// client details didn't match
				err := errors.New(fmt.Sprintf("Host '%s' is already "+
					"in use with different client settings.",
					con.Host))
				go conReq.sendResponse(&ClientResponse{Err: err})
			}
		} else {
			// Set up new client
			con = conReq.Client
			err := con.startClient()
			if nil != err {
				go conReq.sendResponse(&ClientResponse{Err: err})
				continue
			}
			cm.clients[con.Host] = con
			go conReq.sendResponse(&ClientResponse{
				QueryQueue: con.queryQueue,
			})
		}
	}
}
