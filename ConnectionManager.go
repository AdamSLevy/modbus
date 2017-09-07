package modbus

import (
	"errors"
	"fmt"
	"sync"
)

// ConnectionManager is a singleton object that keeps track of clients
// globally for the entire program. Use GetConnectionManager() to get a pointer
// to the global ConnectionManager singleton. Clients are hashed on their
// Host field. Clients are set up and accessed by sending
// ClientRequests to the ConnectionManager goroutine using SendRequest().
type ConnectionManager struct {
	newConnection chan *ConnectionRequest
	deleteClient  chan *string
	clients       map[string]*client
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
			clients:       make(map[string]*client),
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

// requestListener serializes access to the ConnectionManager's clients map by
// listening for incoming ConnectionRequests and delete requests.
// ConnectionRequests will set up a client if necessary and send back a
// ConnectionResponse containing a QueryQueue for the client if successful.
// Delete requests are simply a string containing the Host name of the Client.
// Delete requests are only sent by a
func (cm *ConnectionManager) requestListener() {
	for {
		select {
		case delReq := <-cm.deleteClient:
			delete(cm.clients, *delReq)
		case conReq := <-cm.newConnection:
			cm.handleConnectionRequest(conReq)
		}
	}
}

// handleConnectionRequest
// requestListener listens for incoming ConnectionRequests and sends a
// ConnectionResponse to the ConnectionRequest.Response channel. On success,
// the ConnectionResponse has a valid QueryQueue channel for sending queries to
// the requested client. On failure, ConnectionResponse.Error is set.
// Failure will occur if the client fails or if a client for the
// requested Host already exists with different settings. Existing clients
// can only be requested if all settings match exactly.
func (cm *ConnectionManager) handleConnectionRequest(conReq *ConnectionRequest) {
	if nil == conReq.Response {
		return
	}
	cl, ok := cm.clients[conReq.Host]
	if ok {
		func() {
			cl.wg.Add(1)
			defer cl.wg.Add(-1)
			cl.mu.Lock()
			defer cl.mu.Unlock()

			if cl.ConnectionSettings !=
				conReq.ConnectionSettings {
				// Host is in use but other
				// client details didn't match
				err := fmt.Errorf("Host '%s' is already "+
					"in use with different client "+
					"settings.", cl.Host)
				go conReq.sendResponse(nil, err)
				return
			}

			var run = true
			for run {
				select {
				case delReq := <-cm.deleteClient:
					if *delReq == cl.Host {
						// Restart Client
						qq, err := cl.startClient()
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
			qq, err := cl.newQueryQueue()
			if nil != err {
				go conReq.sendResponse(nil, err)
			} else {
				go conReq.sendResponse(qq, nil)
			}
		}()
	} else {
		// Set up new client
		cl = &client{ConnectionSettings: conReq.ConnectionSettings}
		qq, err := cl.startClient()
		if nil != err {
			go conReq.sendResponse(nil, err)
			return
		}
		cm.clients[cl.Host] = cl
		go conReq.sendResponse(qq, nil)
	}

}
