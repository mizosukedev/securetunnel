package proxy

import (
	"context"
	"errors"
	"net"
	"testing"

	testutil "github.com/mizosukedev/securetunnel/_testutil"
	"github.com/stretchr/testify/suite"
)

var (
	bufSize = 32 * 1024
)

type LocalSocketTest struct {
	suite.Suite
}

func TestLocalSocket(t *testing.T) {
	suite.Run(t, new(LocalSocketTest))
}

// TestNormal confirm the operation when localSocket is used normally.
func (suite *LocalSocketTest) TestNormal() {

	streamID := int32(1)
	serviceID := "serviceID"
	messageFromServer := "message from server"
	messageFromClient := "message from client"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chServerDone := make(chan struct{})
	chClientDone := make(chan struct{})

	listener, err := testutil.StartTCPServer(ctx, func(con net.Conn) {

		socketReader := NewMockSocketReader()
		socket := newLocalSocket(streamID, serviceID, con, socketReader, bufSize)
		socket.Start()

		// server write message
		_, err := socket.Write([]byte(messageFromServer))
		suite.Require().Nil(err)

		// check client message
		actual := <-socketReader.ChOnReadDataArgs
		expected := OnReadDataArgs{
			StreamID:  streamID,
			ServiceID: serviceID,
			Data:      []byte(messageFromClient),
		}

		suite.Require().Equal(expected, actual)

		close(chServerDone)
		<-chClientDone
		socket.Stop()
	})
	defer func() {
		_ = listener.Close()
	}()

	suite.Require().Nil(err)

	con, err := testutil.ConnectTCPServer(listener.Addr().String())

	socketReader := NewMockSocketReader()
	socket := newLocalSocket(streamID, serviceID, con, socketReader, bufSize)
	socket.Start()

	// client write message
	_, err = socket.Write([]byte(messageFromClient))
	suite.Require().Nil(err)

	// check server message
	actual := <-socketReader.ChOnReadDataArgs
	expected := OnReadDataArgs{
		StreamID:  streamID,
		ServiceID: serviceID,
		Data:      []byte(messageFromServer),
	}

	suite.Require().Equal(expected, actual)

	close(chClientDone)
	<-chServerDone

	// Confirm no panic occurs even if Stop() is executed multiple times.
	socket.Stop()
	socket.Stop()

	// socket.Stop() -> con.Read() returns error -> OnReadError is triggered
	onReadErrorArgs := <-socketReader.ChOnReadError
	suite.Require().Equal(streamID, onReadErrorArgs.StreamID)
	suite.Require().Equal(serviceID, onReadErrorArgs.ServiceID)
	suite.Require().NotNil(onReadErrorArgs.Err)
}

// TestWriteError confirm that Write method returns error if writing to socket fails.
func (suite *LocalSocketTest) TestWriteError() {

	streamID := int32(1)
	serviceID := "serviceID"

	mockConn := testutil.NewMockConn()
	mockConn.MockWrite = func(b []byte) (n int, err error) {
		return -1, errors.New("test error")
	}

	mockSocketReader := NewMockSocketReader()

	socket := newLocalSocket(streamID, serviceID, mockConn, mockSocketReader, bufSize)

	_, err := socket.Write([]byte{})
	suite.Require().NotNil(err)
}

// TestOnReadDataError confirm that localSocket stop, if OnReadData returns error.
func (suite *LocalSocketTest) TestOnReadDataError() {

	streamID := int32(1)
	serviceID := "serviceID"
	messageFromServer := "message from server"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := testutil.StartTCPServer(ctx, func(con net.Conn) {

		socketReader := NewMockSocketReader()
		socket := newLocalSocket(streamID, serviceID, con, socketReader, bufSize)
		socket.Start()

		// server write message
		_, err := socket.Write([]byte(messageFromServer))
		suite.Require().Nil(err)

		socket.Stop()
	})
	defer func() {
		_ = listener.Close()
	}()

	suite.Require().Nil(err)

	con, err := testutil.ConnectTCPServer(listener.Addr().String())

	socketReader := NewMockSocketReader()
	// OnReadData returns error
	socketReader.MockOnReadData = func(streamID int32, serviceID string, data []byte) error {
		return errors.New("test error")
	}
	socket := newLocalSocket(streamID, serviceID, con, socketReader, bufSize)
	socket.Start()

	// confirm that goroutine terminates
	<-socket.chTerminate

	socket.Stop()
}
