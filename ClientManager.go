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
	newConnection chan *ConnectionRequest
	deleteClient  chan *string
	clients       map[string]*Client
}

var clientManager *ClientManager
var once sync.Once

// GetClientManager returns a pointer to the singleton instance of
// ClientManager, initializing and starting the ClientManager goroutine
// if necessary.
func GetClientManager() *ClientManager {
	once.Do(func() {
		clientManager = &ClientManager{
			newConnection: make(chan *ConnectionRequest),
			deleteClient:  make(chan *string),
			clients:       make(map[string]*Client),
		}

		go clientManager.requestListener()
	})

	return clientManager
}

// SendRequest sends a ClientRequest to the ClientManager runtime. The
// caller should expect a ClientResponse on the Response channel.
func (cm *ClientManager) SendRequest(req *ConnectionRequest) error {
	if nil == cm.newConnection {
		return errors.New("Uninitialized ClientManager")
	}
	go func() {
		cm.newConnection <- req
	}()
	return nil
}

// requestListener listens for incoming ConnectionRequests and sends a
// ConnectionResponse to the ConnectionRequest.Response channel. On success,
// the ConnectionResponse has a valid QueryQueue channel for sending queries to
// the requested client. On failure, ConnectionResponse.Error is set.
// Failure will occur if the client fails or if a client for the
// requested Host already exists with different settings. Existing clients
// can only be requested if all settings match exactly.
func (cm *ClientManager) requestListener() {
	for {
		select {
		case delReq := <-cm.deleteClient:
			delete(cm.clients, *delReq)
		case conReq := <-cm.newConnection:
			if nil == conReq.Response {
				continue
			}
			cl, ok := cm.clients[conReq.Host]
			if ok {
				func() {
					cl.wg.Add(1)
					defer cl.wg.Add(-1)
					cl.mu.Lock()
					defer cl.mu.Unlock()

					if cl.Connection != conReq.Connection {
						// Host is in use but other
						// client details didn't match
						err := errors.New(fmt.Sprintf("Host '%s' is already "+
							"in use with different client settings.",
							cl.Host))
						go conReq.sendResponse(nil, err)
						return
					}

					var run bool = true
					for run {
						select {
						case delReq := <-cm.deleteClient:
							if *delReq == cl.Host {
								// Restart Client
								qq, err := cl.StartClient()
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
				cl = &Client{Connection: conReq.Connection}
				qq, err := cl.StartClient()
				if nil != err {
					go conReq.sendResponse(nil, err)
					continue
				}
				cm.clients[cl.Host] = cl
				go conReq.sendResponse(qq, nil)
			}
		}
	}
}
