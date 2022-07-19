package proxy

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/mizosukedev/securetunnel/log"
)

// TCPServer is a structure for building TCP server locally.
// This structure cannot be reused.
type TCPServer struct {
	name     string
	network  string
	address  string
	listener net.Listener
	acceptWG *sync.WaitGroup
	chStop   chan struct{}
	mutex    *sync.Mutex
	started  bool
}

// NewTCPServer returns a tcpServer instance.
func NewTCPServer(name string, network string, address string) (*TCPServer, error) {

	_, err := net.ResolveUnixAddr(network, address)
	if err == nil {

		// Delete domain socket file that has already existed.
		_, err = os.Stat(address)
		if err == nil {

			removeErr := os.Remove(address)
			if removeErr != nil {
				removeErr = fmt.Errorf(
					"failed to delete domain socket file. Path=%s: %w",
					address,
					removeErr)
				return nil, removeErr
			}

		}
	}

	listener, err := net.Listen(network, address)
	if err != nil {
		err = fmt.Errorf(
			"failed to start tcp server listening. Network=%s Address=%s: %w",
			network,
			address,
			err)
		return nil, err
	}

	instance := &TCPServer{
		name:     name,
		network:  network,
		address:  address,
		listener: listener,
		acceptWG: &sync.WaitGroup{},
		chStop:   make(chan struct{}, 1),
		mutex:    &sync.Mutex{},
		started:  false,
	}

	return instance, nil
}

// Start accepting in the background.
// When the client connects, execute 'handler' function.
// This method can only be called once.
// The second and subsequent calls do nothing.
func (server *TCPServer) Start(handler func(net.Conn)) {

	server.mutex.Lock()
	defer server.mutex.Unlock()
	if server.started {
		log.Infof(
			"tcp server has already started Name=%s Network=%s Address=%s",
			server.name,
			server.network,
			server.address)
		return
	}
	server.started = true

	// -----------------
	//  first call

	log.Infof(
		"tcp server start accepting. Name=%s Network=%s Address=%s",
		server.name,
		server.network,
		server.address)

	server.acceptWG.Add(1)

	go func() {

		defer server.acceptWG.Done()

		for {

			select {
			case <-server.chStop:
				return
			default:
			}

			con, err := server.listener.Accept()
			if err != nil {
				log.Warnf(
					"tcp server accept error. Name=%s Network=%s Address=%s: %v",
					server.name,
					server.network,
					server.address,
					err)
				continue
			}

			server.acceptWG.Add(1)
			go func() {

				defer server.acceptWG.Done()

				if handler != nil {
					handler(con)
				}

			}()

		}

	}()

}

// Stop accepting.
func (server *TCPServer) Stop() {

	close(server.chStop)

	err := server.listener.Close()
	if err != nil {
		err = fmt.Errorf(
			"close listener error. Name=%s Network=%s Address=%s: %w",
			server.name,
			server.network,
			server.address,
			err)
		log.Error(err)
	}

	// wait for all handlers.
	server.acceptWG.Wait()

	log.Infof(
		"tcp server stopped. Name=%s Network=%s Address=%s",
		server.name,
		server.network,
		server.address)
}
