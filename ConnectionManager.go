package modbus

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// ConnectionRequest holds Connection settings and a Response channel.
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
	ConnectionSettings
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
func (req *ConnectionRequest) sendResponse(q QueryQueue, err error) {
	req.Response <- &ConnectionResponse{
		QueryQueue: q,
		Err:        err,
	}
}

// ConnectionResponse is returned on the Response channel of a previously sent
// ConnectionRequest. On success, Err is nil and the QueryQueue channel can be
// used to send Queries to a Client with the requested Connection.
type ConnectionResponse struct {
	QueryQueue
	Err error
}

// ConnectionManager is a singleton object that keeps track of connections
// globally for the entire program. Use GetConnectionManager() to get a pointer
// to the global ConnectionManager singleton. Clients are hashed on their Host
// field. Clients are set up and accessed by sending ClientRequests to the
// ConnectionManager goroutine using SendRequest().
type ConnectionManager struct {
	newConnection chan *ConnectionRequest
	deleteClient  chan *string
	connections   map[string]*Connection
}

var connectionManager *ConnectionManager
var once sync.Once

// GetConnectionManager returns a pointer to the singleton instance of
// ConnectionManager, initializing and starting the ConnectionManager goroutine
// if necessary.
func GetConnectionManager() *ConnectionManager {
	once.Do(func() {
		connectionManager = &ConnectionManager{
			newConnection: make(chan *ConnectionRequest),
			deleteClient:  make(chan *string),
			connections:   make(map[string]*Connection),
		}

		go connectionManager.requestListener()
	})

	return connectionManager
}

// SendRequest sends a ClientRequest to the ConnectionManager runtime. The
// caller should expect a ClientResponse on the Response channel.
func (cm *ConnectionManager) SendRequest(req *ConnectionRequest) error {
	if nil == cm.newConnection {
		return errors.New("Uninitialized ConnectionManager")
	}
	cm.newConnection <- req
	return nil
}

// requestListener serializes access to the ConnectionManager's connections map by
// listening for incoming ConnectionRequests and delete requests.
// ConnectionRequests will set up a connection if necessary and send back a
// ConnectionResponse containing a QueryQueue for the connection if successful.
// Delete requests are simply a string containing the Host name of the Client.
// Delete requests are only sent by a
func (cm *ConnectionManager) requestListener() {
	for {
		select {
		case delReq := <-cm.deleteClient:
			delete(cm.connections, *delReq)
		case conReq := <-cm.newConnection:
			cm.handleConnectionRequest(conReq)
		}
	}
}

// handleConnectionRequest
// requestListener listens for incoming ConnectionRequests and sends a
// ConnectionResponse to the ConnectionRequest.Response channel. On success,
// the ConnectionResponse has a valid QueryQueue channel for sending queries to
// the requested connection. On failure, ConnectionResponse.Error is set.
// Failure will occur if the connection fails or if a connection for the
// requested Host already exists with different settings. Existing connections
// can only be requested if all settings match exactly.
func (cm *ConnectionManager) handleConnectionRequest(conReq *ConnectionRequest) {
	if nil == conReq.Response {
		return
	}
	con, ok := cm.connections[conReq.Host]
	if ok {
		func() {
			con.wg.Add(1)
			defer con.wg.Add(-1)
			con.mu.Lock()
			defer con.mu.Unlock()

			if con.ConnectionSettings !=
				conReq.ConnectionSettings {
				// Host is in use but other
				// connection details didn't match
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
					delete(cm.connections, *delReq)
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
		// Set up new connection
		con = &Connection{ConnectionSettings: conReq.ConnectionSettings}
		con.isManagedConnection = true
		qq, err := con.Start()
		if nil != err {
			go conReq.sendResponse(nil, err)
			return
		}
		cm.connections[con.Host] = con
		go conReq.sendResponse(qq, nil)
	}

}
