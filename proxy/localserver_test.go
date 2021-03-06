package proxy

import (
	"errors"
	"net"
	"os"
	"runtime"
	"testing"

	testutil "github.com/mizosukedev/securetunnel/_testutil"
	"github.com/stretchr/testify/suite"
)

var (
	socketFilepath = "./.socket.sock"
	freeAddress    = testutil.FreeLocalAddress()
)

type TCPServerTest struct {
	suite.Suite
}

func TestTCPServer(t *testing.T) {
	suite.Run(t, new(TCPServerTest))
}

func (suite *TCPServerTest) BeforeTest(suiteName, testName string) {
	if _, err := os.Stat(socketFilepath); err == nil {
		err = os.Remove(socketFilepath)
		suite.Require().Nil(err)
	}
}

func (suite *TCPServerTest) AfterTest(suiteName, testName string) {
	if _, err := os.Stat(socketFilepath); err == nil {
		err = os.Remove(socketFilepath)
		suite.Require().Nil(err)
	}
}

// TestNormal confirm the operation when tcpServer is used normally.
func (suite *TCPServerTest) TestNormal() {

	configs := []ServiceConfig{
		{"rdp", "tcp", freeAddress},     // TCP
		{"ssh", "unix", socketFilepath}, // domain socket
	}

	for _, config := range configs {

		if config.Network == "unix" && runtime.GOOS == "windows" {
			continue
		}

		server, err := NewTCPServer(config.ServiceID, config.Network, config.Address)
		suite.Require().Nil(err)
		suite.Require().False(server.started)

		clientMessage := "test message"
		chMessage := make(chan string)

		server.Start(func(c net.Conn) {
			buf := make([]byte, len(clientMessage))

			_, readErr := c.Read(buf)
			suite.Require().Nil(readErr)

			chMessage <- string(buf)
		})
		defer server.Stop()

		suite.Require().True(server.started)

		con, err := net.Dial(config.Network, config.Address)
		suite.Require().Nil(err)
		defer con.Close()

		_, err = con.Write([]byte(clientMessage))
		suite.Require().Nil(err)

		receivedMessage := <-chMessage

		suite.Require().Equal(clientMessage, receivedMessage)
	}
}

// TestNewTCPServerSocketFileExists confirm that the instance is created successfully even if socket file exists.
func (suite *TCPServerTest) TestNewTCPServerSocketFileExists() {

	if runtime.GOOS == "windows" {
		return
	}

	file, err := os.Create(socketFilepath)
	suite.Require().Nil(err)

	err = file.Close()
	suite.Require().Nil(err)

	server, err := NewTCPServer("ssh", "unix", socketFilepath)
	suite.Require().Nil(err)

	server.Stop()
	suite.Require().NoFileExists(socketFilepath)
}

// TestNewTCPServerFailToRemoveSocketFile confirm that newTCPServer returns error,
// if socket file deletion fails.
func (suite *TCPServerTest) TestNewTCPServerFailToRemoveSocketFile() {

	if runtime.GOOS == "windows" {
		return
	}

	_, err := NewTCPServer("ssh", "unix", ".")
	suite.Require().NotNil(err)
}

// TestNewTCPServerListenError If net.Listen returns an error, make sure newTCPServer() returns an error.
func (suite *TCPServerTest) TestNewTCPServerListenError() {

	_, err := NewTCPServer("ssh", "hoge", "127.0.0.1:123456")
	suite.Require().NotNil(err)
}

// TestStopCloseError confirm that server stop, if net.Listener.Close returns an error.
func (suite *TCPServerTest) TestStopCloseError() {

	server, err := NewTCPServer("ssh", "tcp", freeAddress)
	suite.Require().Nil(err)

	server.Start(func(c net.Conn) {})

	err = server.listener.Close()
	suite.Require().Nil(err)

	listener := testutil.NewMockListener()
	server.listener = listener

	listener.MockClose = func() error {
		return errors.New("test error")
	}

	server.Stop()
	server.acceptWG.Wait()
}

// TestStartDuplicate confirm that nothing happens after the second time,
// if the start method is executed twice.
func (suite *TCPServerTest) TestStartExecuteTwice() {

	server, err := NewTCPServer("ssh", "tcp", freeAddress)
	suite.Require().Nil(err)

	suite.Require().False(server.started)
	server.Start(func(c net.Conn) {})
	suite.Require().True(server.started)

	restartCount := 5
	for i := 0; i < restartCount; i++ {
		server.Start(func(c net.Conn) {})
	}

	server.Stop()
}
