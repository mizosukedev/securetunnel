package testutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type ReadMessageResult struct {
	MessageType int
	Message     []byte
	Err         error
}

type SecureTunnelServer struct {
	Host           string
	Endpoint       *url.URL
	ChRequest      chan *http.Request
	ChMessage      chan *ReadMessageResult
	ChWebSocket    chan *websocket.Conn
	ChPing         chan string
	RequestHandler func(w http.ResponseWriter, r *http.Request) (endResponse bool)
}

func NewSecureTunnelServer() *SecureTunnelServer {

	address := FreeLocalAddress()
	endpoint := fmt.Sprintf("ws://%s/tunnel", address)
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		panic(err)
	}

	instance := &SecureTunnelServer{
		Host:        address,
		Endpoint:    endpointURL,
		ChRequest:   make(chan *http.Request, 10),
		ChMessage:   make(chan *ReadMessageResult, 10),
		ChWebSocket: make(chan *websocket.Conn, 10),
		ChPing:      make(chan string, 100),
	}

	return instance
}

func (server *SecureTunnelServer) Start(ctx context.Context) {

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/tunnel", server.handleTunnel)

	httpServer := http.Server{
		Addr:    server.Host,
		Handler: serverMux,
	}

	go func() {
		err := httpServer.ListenAndServe()
		fmt.Println(err)
	}()

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(ctx)
	}()

	connected := false
	maxRetry := 5
	for i := 0; i < maxRetry; i++ {
		con, err := net.Dial("tcp", server.Host)
		if err == nil {
			con.Close()
			connected = true
			break
		}

		time.Sleep(time.Millisecond * 200)
	}

	if !connected {
		panic("failed to start http server")
	}
}

func (server *SecureTunnelServer) handleTunnel(w http.ResponseWriter, r *http.Request) {

	server.ChRequest <- r

	if server.RequestHandler != nil {
		endResponse := server.RequestHandler(w, r)
		if endResponse {
			return
		}
	}

	upgrader := websocket.Upgrader{}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic("failed to upgrade websocket")
	}

	ws.SetPingHandler(func(appData string) error {
		server.ChPing <- appData
		return nil
	})

	server.ChWebSocket <- ws

	defer ws.Close()

	for {

		wsMessageType, wsMessageBin, err := ws.ReadMessage()

		server.ChMessage <- &ReadMessageResult{
			MessageType: wsMessageType,
			Message:     wsMessageBin,
			Err:         err,
		}

		if err != nil {
			return
		}
	}

}
