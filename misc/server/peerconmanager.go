package server

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mizosukedev/securetunnel/aws"
	"github.com/mizosukedev/securetunnel/log"
)

// [#67] When making the server a scalable implementation, reimplement it here.

type peerConManager struct {
	rwMutex    *sync.RWMutex
	store      Store
	peerConMap map[peerConnectionKey]*peerConnection
}

func (manager *peerConManager) connect(
	tunnelID string,
	mode aws.Mode,
	con *websocket.Conn,
	serviceIDs []string) error {

	peerKey := peerConnectionKey{
		tunnelID: tunnelID,
		mode:     mode,
	}

	otherPeerKey := peerConnectionKey{
		tunnelID: tunnelID,
		mode:     aws.OtherMode(mode),
	}

	peerConnection := newPeerConnection(con, serviceIDs)

	func() {
		manager.rwMutex.Lock()
		defer manager.rwMutex.Unlock()

		// if peer has been already connected
		existedPeerCon, ok := manager.peerConMap[peerKey]
		if ok {
			err := existedPeerCon.close(
				websocket.CloseProtocolError,
				"overwritten by other connection")
			if err != nil {
				log.Error(err)
			}
		}

		// associate peers
		otherPeerConnection, ok := manager.peerConMap[otherPeerKey]
		if ok {
			peerConnection.associate(otherPeerConnection)
			otherPeerConnection.associate(peerConnection)
		} else {
			// the other peer is not ready yet
			peerConnection.associate(&nilConWriter{})
		}

		manager.peerConMap[peerKey] = peerConnection
	}()

	err := peerConnection.run()

	manager.rwMutex.Lock()
	defer manager.rwMutex.Unlock()

	// If multiple connections of the same tunnel and the same mode are connected,
	// this condition is not met.
	if manager.peerConMap[peerKey] == peerConnection {
		delete(manager.peerConMap, peerKey)
		otherPeerConnection, ok := manager.peerConMap[otherPeerKey]
		if ok {
			otherPeerConnection.associate(&nilConWriter{})
		}
	}

	return err
}

func (manager *peerConManager) closeExpiredConnection() {

	manager.rwMutex.RLock()
	defer manager.rwMutex.RUnlock()

	connectionsMap := map[string][]*peerConnection{}

	for key, peerCon := range manager.peerConMap {
		connections := connectionsMap[key.tunnelID]
		connections = append(connections, peerCon)
		connectionsMap[key.tunnelID] = connections
	}

	for tunnelID, connections := range connectionsMap {

		tunnel, err := manager.store.GetTunnel(tunnelID)
		if err != nil {
			log.Error(err)
		}

		if !tunnel.Open() {
			for _, connection := range connections {
				err := connection.close(
					websocket.CloseNormalClosure,
					aws.StatusReasonTunnelClosed)

				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}
