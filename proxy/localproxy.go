package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/mizosukedev/securetunnel/client"
	"github.com/mizosukedev/securetunnel/log"
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

// LocalProxy represents localproxy in aws secure tunneling service.
type LocalProxy struct {
	mode             client.Mode
	localDialTimeout time.Duration
	awsClient        client.AWSClient
	socketManager    *LocalSocketManager
	serviceMap       map[string]ServiceConfig // [ServiceID]*ServiceConfig
}

// NewLocalProxy returns a LocalProxy instance.
func NewLocalProxy(options LocalProxyOptions) (*LocalProxy, error) {

	// map[ServiceID]
	serviceMap := map[string]ServiceConfig{}

	for _, config := range options.ServiceConfigs {

		serviceMap[config.ServiceID] = config
	}

	localProxy := &LocalProxy{
		mode:             options.Mode,
		localDialTimeout: options.LocalDialTimeout,
		serviceMap:       serviceMap,
	}

	listener := &eventLisnter{
		localProxy: localProxy,
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
func (listener *eventLisnter) OnStreamStart(message *client.Message) error {

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
func (listener *eventLisnter) OnStreamReset(message *client.Message) {

	listener.localProxy.socketManager.StopSocket(message.StreamId)
}

// OnSessionReset Refer to AWSMeesageListener.OnSessionReset
func (listener *eventLisnter) OnSessionReset(message *client.Message) {

	listener.localProxy.socketManager.StopAll()
}

// OnReceivedData Refer to AWSMeesageListener.OnReceivedData
func (listener *eventLisnter) OnReceivedData(message *client.Message) error {

	err := listener.localProxy.socketManager.Write(message.StreamId, message.ServiceId, message.Payload)
	return err
}

// OnReceivedServiceIDs Refer to AWSMeesageListener.OnReceivedServiceIDs
func (listener *eventLisnter) OnReceivedServiceIDs(message *client.Message) error {

	// In source mode, start server listening.

	return nil
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
