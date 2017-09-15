package modbus

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ClientManager is a singleton object that serializes and enforces uniqueness
// on the concurrent creation of clients and associated ClientHandles. Clients
// are uniquely hashed by their ConnectionSettings.Host string. If a client
// with a given Host string is already running, then all ConnectionSettings
// must also match for a ClientHandle to be successfully returned by
// SetupClient.
type ClientManager interface {
	SetupClient(cs ConnectionSettings) (*ClientHandle, error)
}

// clientManager is the underlying concrete type implementing ClientManager.
type clientManager struct {
	newClient    chan *clientRequest
	deleteClient chan *string
	clients      map[string]*client
}

// clntMngr is the pointer to the singleton instance of clientManager.
var clntMngr atomic.Value
var once sync.Once

// unmanagedClients is a flag that protects against using both user managed
// clients created with NewClient, and clients managed by ClientManager.
var unmanagedClients uint32

// GetClientManager sets up the clientManager singleton once and returns the
// ClientManager interface. GetClientManager will fail if NewClient has been
// called previously in the program.
func GetClientManager() (ClientManager, error) {
	if 0 != atomic.LoadUint32(&unmanagedClients) {
		return nil, fmt.Errorf("Cannot start ClientManager after Clients have " +
			"been initialized with NewClient")
	}
	once.Do(func() {
		cm := &clientManager{
			newClient:    make(chan *clientRequest),
			deleteClient: make(chan *string),
			clients:      make(map[string]*client),
		}
		clntMngr.Store(cm)

		go cm.requestListener()
	})

	return clntMngr.Load().(*clientManager), nil
}

func (cm *clientManager) SetupClient(cs ConnectionSettings) (*ClientHandle, error) {
	req := newClientRequest(cs)
	cm.newClient <- req
	res := <-req.response
	return res.ClientHandle, res.Err
}

func (cm *clientManager) requestListener() {
	for {
		select {
		case delReq := <-cm.deleteClient:
			delete(cm.clients, *delReq)
		case conReq := <-cm.newClient:
			cm.handleClientRequest(conReq)
		}
	}
}

func (cm *clientManager) handleClientRequest(clReq *clientRequest) {
	cl, ok := cm.clients[clReq.Host]
	if ok {
		func() {
			cl.wg.Add(1)
			defer cl.wg.Add(-1)
			cl.mu.Lock()
			defer cl.mu.Unlock()

			if cl.ConnectionSettings !=
				clReq.ConnectionSettings {
				// Host is in use but other
				// ConnectionSettings details didn't match
				err := fmt.Errorf("Host '%s' is already "+
					"in use with different connection "+
					"settings.", cl.Host)
				go clReq.sendResponse(nil, err)
				return
			}

			var run = true
			for run {
				select {
				case delReq := <-cm.deleteClient:
					if *delReq == cl.Host {
						// Restart Client
						go clReq.sendResponse(
							cl.NewClientHandle())
						return
					}
					delete(cm.clients, *delReq)
				default:
					run = false
				}
			}
			go clReq.sendResponse(cl.NewClientHandle())
		}()
	} else {
		// Set up new client
		cl := newClient(clReq.ConnectionSettings)
		ch, err := cl.NewClientHandle()
		if err != nil {
			go clReq.sendResponse(nil, err)
			return
		}
		cm.clients[cl.Host] = cl
		go clReq.sendResponse(ch, nil)
	}

}

// clientRequests are sent to ClientManager to get access to a Client.
type clientRequest struct {
	ConnectionSettings
	response chan clientResponse
}

type clientResponse struct {
	*ClientHandle
	Err error
}

// newClientRequest initializes a new clientRequest with a ClientHandle
// channel.
func newClientRequest(cs ConnectionSettings) *clientRequest {
	return &clientRequest{
		ConnectionSettings: cs,
		response:           make(chan clientResponse),
	}
}

// sendResponse is a convenience function used by clientManager's runtime to
// return a ClientHandle for a clientRequest.
func (req *clientRequest) sendResponse(ch *ClientHandle, err error) {
	req.response <- clientResponse{
		ClientHandle: ch,
		Err:          err,
	}
}
