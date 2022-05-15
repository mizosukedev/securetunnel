package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/mizosukedev/securetunnel/client"
	"github.com/mizosukedev/securetunnel/log"
	"github.com/mizosukedev/securetunnel/protomsg"
)

// LocalProxyOptions represents options of LocalProxy.
type LocalProxyOptions struct {

	// Mode represents local proxy mode(ModeSource or ModeDestination).
	Mode client.Mode

	// ServiceConfigs represents the information of the local services to connect to in destination mode,
	// and represents the information of the local services to listen in source mode.
	ServiceConfigs []ServiceConfig

	// Endpoint represents service endpoint.
	Endpoint *url.URL

	// Token represents token in the return value of AWS OpenTunnel WebAPI.
	Token string

	// TLSConfig If you want to customize tls.Config, set the value. Otherwise it sets null.
	TLSConfig *tls.Config

	// EndpointDialTimeout sets the timeout value when connecting to websocket.
	EndpointDialTimeout time.Duration

	// LocalDialTimeout sets the timeout value when connection to local service(ex. ssh server).
	LocalDialTimeout time.Duration

	// EndpointReconnectInterval represents the interval when reconnecting to endpoint.
	EndpointReconnectInterval time.Duration

	// PingInterval represents the interval when sending ping in websocket.
	PingInterval time.Duration
}

// Validate options and returns result.
func (options LocalProxyOptions) Validate() error {

	switch options.Mode {
	case client.ModeDestination, client.ModeSource:
	default:
		err := fmt.Errorf("invalid mode: %v", options.Mode)
		return err
	}

	switch options.Endpoint.Scheme {
	case "ws", "wss":
	default:
		err := errors.New("endpoint scheme should be 'ws://' or 'wss://'")
		return err
	}

	if len(options.ServiceConfigs) == 0 {
		err := errors.New("ServiceConfigs is empty")
		return err
	}

	if options.Endpoint == nil {
		err := errors.New("Endpoint is nil")
		return err
	}

	if options.Token == "" {
		err := errors.New("Token is empty")
		return err
	}

	return nil
}

// LocalProxy represents localproxy in aws secure tunneling service.
type LocalProxy struct {
	mode             client.Mode
	localDialTimeout time.Duration
	awsClient        client.AWSClient
	socketManager    *LocalSocketManager
	serviceMap       map[string]ServiceConfig // [ServiceID]*ServiceConfig
	serverMap        map[string]*tcpServer    // [ServiceID]*tcpServer
}

// NewLocalProxy returns a LocalProxy instance.
func NewLocalProxy(options LocalProxyOptions) (*LocalProxy, error) {

	err := options.Validate()
	if err != nil {
		return nil, err
	}

	// map[ServiceID]
	serviceMap := map[string]ServiceConfig{}
	serverMap := map[string]*tcpServer{}

	for _, config := range options.ServiceConfigs {

		serviceMap[config.ServiceID] = config

		if options.Mode == client.ModeSource {
			server, err := newTCPServer(config)
			if err != nil {
				return nil, err
			}

			serverMap[config.ServiceID] = server
		}
	}

	localProxy := &LocalProxy{
		mode:             options.Mode,
		localDialTimeout: options.LocalDialTimeout,
		serviceMap:       serviceMap,
		serverMap:        serverMap,
	}

	listener := &eventLisnter{
		localProxy: localProxy,
		idGen:      newStreamIDGen(),
	}

	awsClientOptions := client.AWSClientOptions{
		Mode:              options.Mode,
		Endpoint:          options.Endpoint,
		Token:             options.Token,
		TLSConfig:         options.TLSConfig,
		DialTimeout:       options.EndpointDialTimeout,
		ReconnectInterval: options.EndpointReconnectInterval,
		PingInterval:      options.PingInterval,
		MessageListeners:  []client.AWSMessageListener{listener},
		ConnectHandlers:   []func(){listener.OnConnected},
	}

	awsClient, err := client.NewAWSClient(awsClientOptions)
	if err != nil {
		return nil, err
	}

	socketManager := NewLocalSocketManager(listener)

	localProxy.awsClient = awsClient
	localProxy.socketManager = socketManager

	return localProxy, err
}

// Run the fllowing.
// 	[Source mode]
// 	- Start the local server listening using LocalProxyOptions.ServiceConfigs.
// 	- Connect to the tunneling service.
// 	- When the client(ssh etc.) connects, it sends StreamStart to the tunneling service.
// 	- Continue sending messages from client to the tunneling service and messages from the tunneling service to client.
// 	[Destination mode]
// 	- Connect to the tunneling service.
// 	- Connect to the local service using LocalProxyOptions.ServiceConfigs when StreamStart is received.
// 	- Continue sending messages from the local service to the tunneling service and messages from the tunneling service to the local service.
// This method is a blocking method.
// And this medhod does not return control to the caller until the following phenomenons occur.
// 	- Context is done.
// 	- http response status code is 400-499, when connecting to AWS.
// 	- Tunnel is closed.
func (proxy *LocalProxy) Run(ctx context.Context) error {

	// connect to aws.
	err := proxy.awsClient.Run(ctx)

	// cleanup local connections.
	proxy.socketManager.StopAll()

	return err
}

// eventListener is a struct that implements AWSMessageListener, SocketReader
// and a few other interfaces.
type eventLisnter struct {
	localProxy *LocalProxy
	idGen      *streamIDGen
}

// --------------------------------------------
//  AWSClient OnConnected
// --------------------------------------------

// OnConnected is a method that is executed when the connection to AWS is successful.
func (listener *eventLisnter) OnConnected() {

	// Drops the existing connections when reconnecting.
	listener.localProxy.socketManager.StopAll()
}

// --------------------------------------------
//  AWSMessageListener members
//  Fired when messages are received from AWS
// --------------------------------------------

// OnStreamStart Refer to AWSMeesageListener.OnStreamStart
func (listener *eventLisnter) OnStreamStart(message *protomsg.Message) error {

	svcConfig, ok := listener.localProxy.serviceMap[message.ServiceId]
	if !ok {
		err := fmt.Errorf("unknown serviceID. serviceID=%s", message.ServiceId)
		return err
	}

	con, err := net.DialTimeout(
		svcConfig.Network,
		svcConfig.Address,
		listener.localProxy.localDialTimeout)

	if err != nil {
		err = fmt.Errorf("failed to connect to local service. service=%v: %w", svcConfig, err)
		return err
	}

	err = listener.localProxy.socketManager.StartSocket(message.StreamId, message.ServiceId, con)

	return err
}

// OnStreamReset Refer to AWSMeesageListener.OnStreamReset
func (listener *eventLisnter) OnStreamReset(message *protomsg.Message) {

	listener.localProxy.socketManager.StopSocket(message.StreamId)
}

// OnSessionReset Refer to AWSMeesageListener.OnSessionReset
func (listener *eventLisnter) OnSessionReset(message *protomsg.Message) {

	listener.localProxy.socketManager.StopAll()
}

// OnData Refer to AWSMeesageListener.OnData
func (listener *eventLisnter) OnData(message *protomsg.Message) error {

	err := listener.localProxy.socketManager.Write(message.StreamId, message.ServiceId, message.Payload)
	return err
}

// OnServiceIDs Refer to AWSMeesageListener.OnServiceIDs
func (listener *eventLisnter) OnServiceIDs(message *protomsg.Message) error {

	if listener.localProxy.mode == client.ModeSource {

		for _, server := range listener.localProxy.serverMap {

			serviceID := server.config.ServiceID

			// Check if the service is available.
			available := false
			for _, availableServiceID := range message.AvailableServiceIds {
				if availableServiceID == serviceID {
					available = true
					break
				}
			}

			if !available {
				log.Warnf(
					"'%s' is not available service. You need to specify that service when you run OpenTunnel WebAPI.",
					serviceID)
				continue
			}

			// start accepting
			onConnected := listener.onConnectedHandler(serviceID)

			// The second and subsequent calls do nothing.
			server.Start(onConnected)
		}

	}

	return nil
}

// --------------------------------------------
//  LocalServer handler
// --------------------------------------------

// onConnectedHandler returns the handler to be executed when connecting to the local server.
func (listener *eventLisnter) onConnectedHandler(serviceID string) func(net.Conn) {

	return func(con net.Conn) {

		streamID := listener.idGen.generate()

		err := listener.localProxy.awsClient.SendStreamStart(streamID, serviceID)
		if err != nil {
			log.Error(err)

			err = con.Close()
			if err != nil {
				log.Error(err)
			}

			return
		}

		err = listener.localProxy.socketManager.StartSocket(streamID, serviceID, con)
		if err != nil {
			log.Error(err)

			err = con.Close()
			if err != nil {
				log.Error(err)
			}
		}

	}

}

// --------------------------------------------
//  SocketReader members
//  Fired when datas are read from local socket.
// --------------------------------------------

// OnReadData Refer to SocketReader.OnReadData
func (listener *eventLisnter) OnReadData(
	streamID int32,
	serviceID string,
	data []byte) error {

	err := listener.localProxy.awsClient.SendData(streamID, serviceID, data)
	return err
}

// OnReadError Refer to SocketReader.OnReadError
func (listener *eventLisnter) OnReadError(
	streamID int32,
	serviceID string,
	err error) {

	resetErr := listener.localProxy.awsClient.SendStreamReset(streamID, serviceID)
	if resetErr != nil {
		log.Error(resetErr)
	}
}
