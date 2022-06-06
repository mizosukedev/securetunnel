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
	config   ServiceConfig
	listener net.Listener
	acceptWG *sync.WaitGroup
	chStop   chan struct{}
	mutex    *sync.Mutex
	started  bool
}

// NewTCPServer returns a tcpServer instance.
func NewTCPServer(config ServiceConfig) (*TCPServer, error) {

	_, err := net.ResolveUnixAddr(config.Network, config.Address)
	if err == nil {

		// Delete domain socket file that has already existed.
		_, err = os.Stat(config.Address)
		if err == nil {

			err := os.Remove(config.Address)
			if err != nil {
				err = fmt.Errorf(
					"failed to delete domain socket file. Path=%s: %w",
					config.Address,
					err)
				return nil, err
			}

		}
	}

	listener, err := net.Listen(config.Network, config.Address)
	if err != nil {
		err = fmt.Errorf(
			"failed to start tcp server listening. ServiceID=%s Network=%s Address=%s: %w",
			config.ServiceID,
			config.Network,
			config.Address,
			err)
		return nil, err
	}

	instance := &TCPServer{
		config:   config,
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
	if server.started {
		log.Infof(
			"tcp server has already started ServiceID=%s Network=%s Address=%s",
			server.config.ServiceID,
			server.config.Network,
			server.config.Address)
		return
	}
	server.started = true
	server.mutex.Unlock()

	// -----------------
	//  first call

	log.Infof(
		"tcp server start accepting. ServiceID=%s Network=%s Address=%s",
		server.config.ServiceID,
		server.config.Network,
		server.config.Address)

	server.acceptWG.Add(1)

	go func() {

		defer server.acceptWG.Done()

		for {

			select {
			case <-server.chStop:
				return

			default:

				con, err := server.listener.Accept()
				if err != nil {
					log.Warnf(
						"tcp server accept error. ServiceID=%s Network=%s Address=%s: %v",
						server.config.ServiceID,
						server.config.Network,
						server.config.Address,
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
		}

	}()

}

// Stop accepting.
func (server *TCPServer) Stop() {

	close(server.chStop)

	err := server.listener.Close()
	if err != nil {
		err = fmt.Errorf(
			"close listener error. ServiceID=%s Network=%s Address=%s: %w",
			server.config.ServiceID,
			server.config.Network,
			server.config.Address,
			err)
		log.Error(err)
	}

	// wait for all handlers.
	server.acceptWG.Wait()

	log.Infof(
		"tcp server stopped. ServiceID=%s Network=%s Address=%s",
		server.config.ServiceID,
		server.config.Network,
		server.config.Address)
}
