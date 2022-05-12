package proxy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"

	testutil "github.com/mizosukedev/securetunnel/_testutil"
	"github.com/stretchr/testify/suite"
)

type LocalSocketManagerTest struct {
	suite.Suite
}

func TestLocalSocketManager(t *testing.T) {
	suite.Run(t, new(LocalSocketManagerTest))
}

// TestNormal confirm the operation when localSocket is used normally.
func (suite *LocalSocketManagerTest) TestNormal() {

	connectCount := 5

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	socketReader := NewMockSocketReader()
	socketManager := NewLocalSocketManager(socketReader)

	listener, err := testutil.StartTCPServer(ctx, func(con net.Conn) {
		readBuf := make([]byte, 1024)
		readSize, err := con.Read(readBuf)
		suite.Require().Nil(err)

		// write the data read from the client as it is.
		readData := readBuf[:readSize]
		_, err = con.Write(readData)
		suite.Require().Nil(err)
	})
	defer listener.Close()

	suite.Require().Nil(err)

	// connect multiple times
	for i := 0; i < connectCount; i++ {
		con, err := testutil.ConnectTCPServer(listener.Addr().String())
		suite.Require().Nil(err)

		serviceID := strconv.Itoa(i)
		err = socketManager.StartSocket(int32(i), serviceID, con)
		suite.Require().Nil(err)
	}

	// client write message
	for i := 0; i < connectCount; i++ {

		serviceID := strconv.Itoa(i)
		clientMessage := fmt.Sprintf("StreamID=%d, ServiceID=%s", i, serviceID)

		err = socketManager.Write(int32(i), serviceID, []byte(clientMessage))
		suite.Require().Nil(err)
	}

	// check server message
	for i := 0; i < connectCount; i++ {

		actual := <-socketReader.ChOnReadDataArgs

		streamID := int32(0)
		serviceID := ""
		_, err := fmt.Sscanf(string(actual.Data), "StreamID=%d, ServiceID=%s", &streamID, &serviceID)
		suite.Require().Nil(err)

		expected := OnReadDataArgs{
			StreamID:  streamID,
			ServiceID: serviceID,
			Data:      []byte(fmt.Sprintf("StreamID=%d, ServiceID=%s", streamID, serviceID)),
		}

		suite.Require().Equal(expected, actual)
	}

	// Stop one socket.
	socketManager.StopSocket(1)
	_, ok := socketManager.socketMap[1]
	suite.Require().False(ok)

	// Stop all sockets.
	socketManager.StopAll()
	suite.Require().Len(socketManager.socketMap, 0)
}

// TestStartSocketStreamIDExists confirm that the error occurs when the StreamID of the argument of StartSocket() already exists.
func (suite *LocalSocketManagerTest) TestStartSocketStreamIDExists() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	socketReader := NewMockSocketReader()
	socketManager := NewLocalSocketManager(socketReader)

	listener, err := testutil.StartTCPServer(ctx, func(con net.Conn) {
	})
	defer listener.Close()

	suite.Require().Nil(err)

	con, err := testutil.ConnectTCPServer(listener.Addr().String())
	suite.Require().Nil(err)

	err = socketManager.StartSocket(int32(10), "serviceID", con)
	suite.Require().Nil(err)

	// duplicate StreamID
	err = socketManager.StartSocket(int32(10), "serviceID", con)
	suite.Require().NotNil(err)
}

// TestWriteStreamIDNotFound confirm that the error occurs when the StreamID of the argument of Write() does not exist.
func (suite *LocalSocketManagerTest) TestWriteStreamIDNotFound() {

	socketReader := NewMockSocketReader()
	socketManager := NewLocalSocketManager(socketReader)

	err := socketManager.Write(int32(10), "serviceID", []byte{})
	suite.Require().NotNil(err)
}
