package modbus

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// ClientRequests are sent to ClientManager to get access to a Client.
type ClientRequest struct {
	ConnectionSettings
	Response chan *ClientResponse
}

// NewClientRequest initializes a new ClientRequest with a ClientResponse
// channel.
func NewClientRequest() *ClientRequest {
	return &ClientRequest{Response: make(chan *ClientResponse)}
}

// sendResponse is a convenience function used by clientManager's runtime to
// return a ClientResponse for a ClientRequest.
func (req *ClientRequest) sendResponse(q chan Query, err error) {
	req.Response <- &ClientResponse{
		QueryQueue: q,
		Err:        err,
	}
}

// ClientResponse is returned with a valid Query channel
type ClientResponse struct {
	QueryQueue chan Query
	Err        error
}

type ClientManager interface {
	SendRequest(req *ClientRequest) error
}

type clientManager struct {
	newClient    chan *ClientRequest
	deleteClient chan *string
	clients      map[string]*Client
}

var clntMngr *clientManager
var once sync.Once

func GetClientManager() ClientManager {
	once.Do(func() {
		clntMngr = &clientManager{
			newClient:    make(chan *ClientRequest),
			deleteClient: make(chan *string),
			clients:      make(map[string]*Client),
		}

		go clntMngr.requestListener()
	})

	return clntMngr
}

func (cm *clientManager) SendRequest(req *ClientRequest) error {
	if nil == cm.newClient {
		return errors.New("Uninitialized ClientManager")
	}
	cm.newClient <- req
	return nil
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

func (cm *clientManager) handleClientRequest(conReq *ClientRequest) {
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
