package client

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	queryKeyProxyMode    = "local-proxy-mode"
	subProtocolV2        = "aws.iot.securetunneling-2.0"
	headerKeyAccessToken = "access-token"
)

var (
	subProtocols = []string{
		subProtocolV2,
	}
)

// AWSMessageListener is an interface representing event handlers to be fired
// when localproxy received message from secure tunneling service.
type AWSMessageListener interface {

	// OnStreamStart is an event handler that fires when a StreamStart message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#streamstart
	OnStreamStart(message *Message) error

	// OnStreamReset is an event handler that fires when a StreamReset message is received.
	// This method may be executed multiple times with the same stream ID.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#streamreset
	OnStreamReset(message *Message)

	// OnSessionReset is an event handler that fires when a SessionReset message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#sessionreset
	OnSessionReset(message *Message)

	// OnReceivedData is an event handler that fires when a Data message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#data
	OnReceivedData(message *Message) error

	// OnReceivedServiceIDs is an event handler that fires when a ServiceIDs message is received.
	// 	Note:
	// 		The server will also send a ServiceIDs message when reconnecting.
	// 		That is, this method will also be executed when reconnecting.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#serviceids
	OnReceivedServiceIDs(message *Message) error
}

// AWSClient is an interface, for the purpose of connectiong secure tunneling service.
type AWSClient interface {

	// Run keep websocket connection to secure tunneling service.
	// If the connection is successful, execute the following processing.
	// 	- Periodically send a ping frame to the service.
	// 	- Keep reading messages from the service and fire event handlers associated with AWSClientOptions.MessageListeners.
	// This method is a blocking method.
	// And this medhod does not return control to the caller until the following phenomenons occur.
	// 	- Caller context is done.
	// 	- http response status code is 400-499, when connecting to the service.
	// 	- Tunnel is closed.
	Run(ctx context.Context) error

	// SendStreamStart send StreamStart message to the service.
	// This method must be executed after Run() method.
	SendStreamStart(streamID int32, serviceID string) error

	// SendStreamReset send StreamReset message to the service.
	// This method must be executed after Run() method.
	SendStreamReset(streamID int32, serviceID string) error

	// SendData send Data message to the service.
	// This method must be executed after Run() method.
	SendData(streamID int32, serviceID string, data []byte) error
}

// AWSClientOptions is options of AWSClient.
type AWSClientOptions struct {

	// Mode represents local proxy mode.
	Mode Mode

	// Endpoint represents service endpoint.
	Endpoint *url.URL

	// Token represents token in the return value of AWS OpenTunnel WebAPI.
	Token string

	// TLSConfig If you want to customize tls.Config, set the value. Otherwise it sets null.
	TLSConfig *tls.Config

	// DialTimeout sets the timeout value when connecting to websocket.
	DialTimeout time.Duration

	// ReconnectInterval represents the interval when reconnecting.
	ReconnectInterval time.Duration

	// PingInterval represents the interval when sending ping.
	PingInterval time.Duration

	// MessageListeners represents instances which implement AWSMessageListener interface.
	MessageListeners []AWSMessageListener

	// ConnectHandlers are event handlers to be fired
	// when the connection to the service is successful.
	// This event handlers also are fired on reconnection.
	// TODO: think arguments.
	ConnectHandlers []func()
}

// NewAWSClient returns a instance which implements AWSClient.
func NewAWSClient(options AWSClientOptions) (AWSClient, error) {

	// clone url
	endpoint, _ := url.Parse(options.Endpoint.String())

	// create query parameter -> wss://xxx.xxx/xxx?local-proxy-mode=destination or source
	query := endpoint.Query()
	query.Add(queryKeyProxyMode, string(options.Mode))
	endpoint.RawQuery = query.Encode()

	tlsConfig := options.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}

	cloneDefaultDialer := *websocket.DefaultDialer
	dialer := &cloneDefaultDialer
	dialer.TLSClientConfig = tlsConfig
	dialer.Subprotocols = subProtocols

	requestHeader := http.Header{
		headerKeyAccessToken: []string{options.Token},
	}

	instance := &awsClient{
		mode:              options.Mode,
		endpoint:          endpoint,
		token:             options.Token,
		dialTimeout:       options.DialTimeout,
		reconnectInterval: options.ReconnectInterval,
		pingInterval:      options.PingInterval,
		messageListeners:  options.MessageListeners,
		connectHandlers:   options.ConnectHandlers,
		dialer:            dialer,
		requestHeader:     requestHeader,
	}

	return instance, nil
}

// awsClient is a structure that implements AWSClient interface.
type awsClient struct {
	mode              Mode
	endpoint          *url.URL
	token             string
	dialTimeout       time.Duration
	reconnectInterval time.Duration
	pingInterval      time.Duration
	messageListeners  []AWSMessageListener
	connectHandlers   []func()
	dialer            *websocket.Dialer
	requestHeader     http.Header
	con               *websocket.Conn
}

// Run Refer to AWSClient.
func (client *awsClient) Run(ctx context.Context) error {

	return nil
}

// SendStreamStart Refer to AWSClient.
func (client *awsClient) SendStreamStart(streamID int32, serviceID string) error {

	return nil
}

// SendStreamReset Refer to AWSClient.
func (client *awsClient) SendStreamReset(streamID int32, serviceID string) error {

	return nil
}

// SendData Refer to AWSClient.
func (client *awsClient) SendData(streamID int32, serviceID string, data []byte) error {

	return nil
}
