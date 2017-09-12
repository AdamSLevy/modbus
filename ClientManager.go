package modbus

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// ClientRequest holds Connection settings and a Response channel.
// ClientRequests are sent to ClientManager's runtime to get a
// ClientResponse back on the Response channel containing a QueryQueue that
// can be used to send asynchronous Modbus Queries to a Client with the
// specified Client. Upon receiving a ClientRequest the ClientManager
// checks if a Client with the same Client.Host name already exists. If an
// existing Client is found, all other ConnectionSettings must match for a
// successful ClientResponse. If no such Client is found, a new Client is
// created if the Client is set up successfully. A ClientResponse will
// always be returned to the caller on the Response channel with either a
// QueryQueue channel or an error.
type ClientRequest struct {
	ConnectionSettings
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
func (req *ClientRequest) sendResponse(q QueryQueue, err error) {
	req.Response <- &ClientResponse{
		QueryQueue: q,
		Err:        err,
	}
}

// ClientResponse is returned on the Response channel of a previously sent
// ClientRequest. On success, Err is nil and the QueryQueue channel can be
// used to send Queries to a Client with the requested ConnectionSettings.
type ClientResponse struct {
	QueryQueue
	Err error
}

// ClientManager is a singleton object that keeps track of clients
// globally for the entire program. Use GetClientManager() to get a pointer
// to the global ClientManager singleton. Clients are hashed on their Host
// field. Clients are set up and accessed by sending ClientRequests to the
// ClientManager goroutine using SendRequest().
type ClientManager struct {
	newClient    chan *ClientRequest
	deleteClient chan *string
	clients      map[string]*Client
}

var clientManager *ClientManager
var once sync.Once

// GetClientManager returns a pointer to the singleton instance of
// ClientManager, initializing and starting the ClientManager goroutine
// if necessary.
func GetClientManager() *ClientManager {
	once.Do(func() {
		clientManager = &ClientManager{
			newClient:    make(chan *ClientRequest),
			deleteClient: make(chan *string),
			clients:      make(map[string]*Client),
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
	cm.newClient <- req
	return nil
}

// requestListener serializes access to the ClientManager's clients map by
// listening for incoming ClientRequests and delete requests.
// ClientRequests will set up a client if necessary and send back a
// ClientResponse containing a QueryQueue for the client if successful.
// Delete requests are simply a string containing the Host name of the Client.
// Delete requests are only sent by a
func (cm *ClientManager) requestListener() {
	for {
		select {
		case delReq := <-cm.deleteClient:
			delete(cm.clients, *delReq)
		case conReq := <-cm.newClient:
			cm.handleClientRequest(conReq)
		}
	}
}

// handleClientRequest
// requestListener listens for incoming ClientRequests and sends a
// ClientResponse to the ClientRequest.Response channel. On success,
// the ClientResponse has a valid QueryQueue channel for sending queries to
// the requested client. On failure, ClientResponse.Error is set.
// Failure will occur if the client fails or if a client for the
// requested Host already exists with different settings. Existing clients
// can only be requested if all ConnectionSettings match exactly.
func (cm *ClientManager) handleClientRequest(conReq *ClientRequest) {
	if nil == conReq.Response {
		return
	}
	con, ok := cm.clients[conReq.Host]
	if ok {
		func() {
			con.wg.Add(1)
			defer con.wg.Add(-1)
			con.mu.Lock()
			defer con.mu.Unlock()

			if con.ConnectionSettings !=
				conReq.ConnectionSettings {
				// Host is in use but other
				// ConnectionSettings details didn't match
				err := fmt.Errorf("Host '%s' is already "+
					"in use with different connection "+
					"settings.", con.Host)
				go conReq.sendResponse(nil, err)
				return
			}

			var run = true
			for run {
				select {
				case delReq := <-cm.deleteClient:
					if *delReq == con.Host {
						// Restart Client
						qq, err := con.Start()
						if nil != err {
							go conReq.sendResponse(nil, err)
						} else {
							go conReq.sendResponse(qq, nil)
						}
						return
					}
					delete(cm.clients, *delReq)
				default:
					run = false
				}
			}
			qq := con.newQueryQueue()
			if nil == qq {
				log.Fatal("Client is not running")
			} else {
				go conReq.sendResponse(qq, nil)
			}
		}()
	} else {
		// Set up new client
		con = &Client{ConnectionSettings: conReq.ConnectionSettings}
		con.isManagedClient = true
		qq, err := con.Start()
		if nil != err {
			go conReq.sendResponse(nil, err)
			return
		}
		cm.clients[con.Host] = con
		go conReq.sendResponse(qq, nil)
	}

}
