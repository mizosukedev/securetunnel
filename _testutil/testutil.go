// testutil is a package that implements a set of functions
// for testing.
package testutil

import (
	"context"
	"fmt"
	"net"
	"time"
)

// FreeLocalAddress returns free local address.
// -> 127.0.0.1:<free port>
func FreeLocalAddress() string {

	port := FreePort()
	address := fmt.Sprintf("127.0.0.1:%d", port)

	return address
}

// FreePort returns free port.
func FreePort() uint16 {

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	port := uint16(0)
	fmt.Sscanf(listener.Addr().String(), "127.0.0.1:%d", &port)

	return port
}

// StartTCPServer start tcp server.
func StartTCPServer(ctx context.Context, handler func(net.Conn)) (net.Listener, error) {

	address := FreeLocalAddress()

	listener, err := net.Listen("tcp", address)

	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	go func() {

		con, err := listener.Accept()
		if err != nil {
			return
		}

		go handler(con)
	}()

	return listener, nil
}

func ConnectTCPServer(address string) (net.Conn, error) {

	var err error

	for i := 0; i < 5; i++ {
		con, err := net.Dial("tcp", address)
		if err == nil {
			return con, nil
		}

		time.Sleep(time.Millisecond * 500)
	}

	return nil, err
}
