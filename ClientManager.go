package modbus

import (
	"fmt"
	"log"
	"sync"
)

// clientRequests are sent to ClientManager to get access to a Client.
type clientRequest struct {
	ConnectionSettings
	response chan *clientResponse
}

// newClientRequest initializes a new clientRequest with a clientResponse
// channel.
func newClientRequest(cs ConnectionSettings) *clientRequest {
	return &clientRequest{
		ConnectionSettings: cs,
		response:           make(chan *clientResponse),
	}
}

// sendResponse is a convenience function used by clientManager's runtime to
// return a clientResponse for a clientRequest.
func (req *clientRequest) sendResponse(q chan Query, err error) {
	req.response <- &clientResponse{
		QueryQueue: q,
		Err:        err,
	}
}

// clientResponse is returned by clientManager's runtime with an initialized
// Query channel, QueryQueue, if the corresponding clientRequest was
// successful. Otherwise the error, Err, is set.
type clientResponse struct {
	QueryQueue chan Query
	Err        error
}

// ClientManager is a singleton object that uniquely sets up Clients based on
// clientRequests and returns a unique channel for sending Queries to the
// Client encapsulated in a clientResponse. Clients are uniquely hashed by
// their ConnectionSettings.Host string and all additional ConnectionSettings
// must also match for clientRequests for existing Clients to be successful.
type ClientManager interface {
	SetupClient(cs ConnectionSettings) (chan Query, error)
}

// clientManager is the underlying concrete type implementing ClientManager.
type clientManager struct {
	newClient    chan *clientRequest
	deleteClient chan *string
	clients      map[string]*Client
}

// clntMngr is the pointer to the clientManager singleton instance
var clntMngr *clientManager
var once sync.Once

// GetClientManager sets up the clientManager singleton once and returns the
// ClientManager interface.
func GetClientManager() ClientManager {
	once.Do(func() {
		clntMngr = &clientManager{
			newClient:    make(chan *clientRequest),
			deleteClient: make(chan *string),
			clients:      make(map[string]*Client),
		}

		go clntMngr.requestListener()
	})

	return clntMngr
}

func (cm *clientManager) SetupClient(cs ConnectionSettings) (chan Query, error) {
	req := newClientRequest(cs)
	cm.newClient <- req
	res := <-req.response
	return res.QueryQueue, res.Err
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
						qq, err := cl.Start()
						if nil != err {
							go clReq.sendResponse(nil, err)
						} else {
							go clReq.sendResponse(qq, nil)
						}
						return
					}
					delete(cm.clients, *delReq)
				default:
					run = false
				}
			}
			qq := cl.newQueryQueue()
			if nil == qq {
				log.Fatal("Client is not running")
			} else {
				go clReq.sendResponse(qq, nil)
			}
		}()
	} else {
		// Set up new client
		cl = &Client{ConnectionSettings: clReq.ConnectionSettings}
		cl.isManagedClient = true
		qq, err := cl.Start()
		if nil != err {
			go clReq.sendResponse(nil, err)
			return
		}
		cm.clients[cl.Host] = cl
		go clReq.sendResponse(qq, nil)
	}

}
