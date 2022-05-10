package proxy

import (
	"fmt"
	"net"

	"github.com/mizosukedev/securetunnel/log"
)

// SocketReader is an interface that processes data read from local sockets.
type SocketReader interface {
	// OnReadData is an event handler that is fired when reading data from socket.
	// If this method returns error, localSocket will stop.
	OnReadData(streamID int32, serviceID string, data []byte) error
	// OnReadError is an event handler that is fired when it fails to read data from socket.
	OnReadError(streamID int32, serviceID string, err error)
}

// localSocket is a structure that continue reading the data read from a local socket.
// Each time the data is read, the method of the SocketReader interface is triggered.
// This structure manages only one connection.
type localSocket struct {
	streamID      int32
	serviceID     string
	con           net.Conn
	socketReader  SocketReader
	readBufSize   int
	chanTerminate chan struct{}
}

// newLocalSocket returns a localSocket instance.
func newLocalSocket(
	streamID int32,
	serviceID string,
	con net.Conn,
	socketReader SocketReader,
	bufSize int) *localSocket {

	instance := &localSocket{
		streamID:     streamID,
		serviceID:    serviceID,
		con:          con,
		socketReader: socketReader,
		readBufSize:  bufSize,
	}

	return instance
}

// Start to continue reading data from a local socket and sending the data
// via inner SocketReader in the background.
func (socket *localSocket) Start() {

	socket.chanTerminate = make(chan struct{})

	// Start thread for sucking data.
	go func() {
		defer close(socket.chanTerminate)

		err := socket.suck()
		if err != nil {
			log.Error(err)
		}

		err = socket.close()
		if err != nil {
			log.Error(err)
		}
	}()

}

func (socket *localSocket) suck() error {

	for {
		// read
		readBuf := make([]byte, socket.readBufSize)
		readSize, err := socket.con.Read(readBuf)

		if err != nil {
			err = fmt.Errorf(
				"failed to read data from local socket. StreamID=%d ServiceID=%s Addr=%v: %w",
				socket.streamID,
				socket.serviceID,
				socket.con.RemoteAddr(),
				err)

			socket.socketReader.OnReadError(socket.streamID, socket.serviceID, err)
			return err
		}

		// write
		err = socket.socketReader.OnReadData(
			socket.streamID,
			socket.serviceID,
			readBuf[:readSize])

		if err != nil {
			return err
		}
	}
}

// Write data to local socket.
func (socket *localSocket) Write(p []byte) (n int, err error) {

	n, err = socket.con.Write(p)
	if err != nil {
		err = fmt.Errorf(
			"failed to write data to local socket. StreamID=%d ServiceID=%s Addr=%v: %w",
			socket.streamID,
			socket.serviceID,
			socket.con.RemoteAddr(),
			err)
	}

	return n, err
}

// close local socket connection.
// close -> read error -> Socket.OnReadError -> StreamReset -> goroutine terminate
func (socket *localSocket) close() error {

	err := socket.con.Close()
	if err != nil {
		err = fmt.Errorf(
			"failed to close local connection. StreamID=%d ServiceID=%s Addr=%v: %w",
			socket.streamID,
			socket.serviceID,
			socket.con.RemoteAddr(),
			err)
		return err
	}

	return nil
}

// Stop goroutine for reading.
func (socket *localSocket) Stop() {

	err := socket.close()
	if err != nil {
		log.Error(err)
	}

	<-socket.chanTerminate
}
