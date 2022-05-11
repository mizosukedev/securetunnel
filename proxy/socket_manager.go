package proxy

import (
	"fmt"
	"net"
	"sync"
)

const (
	readBufSize = 32 * 1024
)

// LocalSocketManager is a structure for managing local connections.
// In destination mode, it manages connections to local services.
// In source mode, it manages the connection to the client application.
type LocalSocketManager struct {
	rwMutex      *sync.RWMutex
	socketMap    map[int32]*localSocket
	socketReader SocketReader
}

// NewLocalSocketManager returns a LocalSocketManager instance.
func NewLocalSocketManager(socketReader SocketReader) *LocalSocketManager {

	instance := &LocalSocketManager{
		rwMutex:      &sync.RWMutex{},
		socketMap:    map[int32]*localSocket{},
		socketReader: socketReader,
	}

	return instance
}

// StartSocket start reading data and sending data via inner SocketReader.
func (manager *LocalSocketManager) StartSocket(
	streamID int32,
	serviceID string,
	con net.Conn) error {

	socket := newLocalSocket(streamID, serviceID, con, manager.socketReader, readBufSize)

	manager.rwMutex.Lock()
	defer manager.rwMutex.Unlock()

	if _, ok := manager.socketMap[streamID]; ok {
		err := fmt.Errorf("specified streamID already exists. StreamID=%d", streamID)
		return err
	}

	manager.socketMap[streamID] = socket

	socket.Start()

	return nil
}

// Write the data to the appropriate localSocket.
func (manager *LocalSocketManager) Write(streamID int32, serviceID string, data []byte) error {

	manager.rwMutex.RLock()
	socket, ok := manager.socketMap[streamID]
	manager.rwMutex.RUnlock()

	if !ok {
		return fmt.Errorf("unknown streamID. streamID=%d", streamID)
	}

	_, err := socket.Write(data)

	return err
}

// StopSocket the specified LocalSocket.
// This method may be executed multiple times with the same stream ID.
// And the corresponding LocalSocket may have already terminated.
func (manager *LocalSocketManager) StopSocket(streamID int32) {

	manager.rwMutex.Lock()
	defer manager.rwMutex.Unlock()

	socket, ok := manager.socketMap[streamID]
	if ok {
		socket.Stop()
		delete(manager.socketMap, streamID)
	}
}

// StopAll stop all LocalSocket and wait for all LocalSocket to terminate.
func (manager *LocalSocketManager) StopAll() {

	manager.rwMutex.Lock()
	defer manager.rwMutex.Unlock()

	for streamID, socket := range manager.socketMap {
		socket.Stop()
		delete(manager.socketMap, streamID)
	}
}
