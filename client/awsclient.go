package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mizosukedev/securetunnel/log"
	"github.com/mizosukedev/securetunnel/protomsg"
	"google.golang.org/protobuf/proto"
)

const (
	queryKeyProxyMode        = "local-proxy-mode"
	subProtocolV2            = "aws.iot.securetunneling-2.0"
	headerKeyAccessToken     = "access-token"
	headerKeyStatusReason    = "X-Status-Reason"
	statusReasonTunnelClosed = "Tunnel is closed"
	sizeOfMessageSize        = 2
	maxWebSocketFrameSize    = 131076
	pingTimeout              = time.Second * 3

	channelBufSizePerStreamID = 10
)

var (
	subProtocols = []string{
		subProtocolV2,
	}

	ErrTunnelClosed = errors.New("Tunnel is closed")
)

// AWSMessageListener is an interface representing event handlers to be fired
// when localproxy received message from secure tunneling service.
// AWSClient generates one goroutine for each StreamID to process the message.
type AWSMessageListener interface {

	// OnStreamStart is an event handler that fires when a StreamStart message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#streamstart
	OnStreamStart(message *protomsg.Message) error

	// OnStreamReset is an event handler that fires when a StreamReset message is received.
	// This method may be executed multiple times with the same stream ID.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#streamreset
	OnStreamReset(message *protomsg.Message)

	// OnSessionReset is an event handler that fires when a SessionReset message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#sessionreset
	OnSessionReset(message *protomsg.Message)

	// OnData is an event handler that fires when a Data message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#data
	OnData(message *protomsg.Message) error

	// OnServiceIDs is an event handler that fires when a ServiceIDs message is received.
	// 	Note:
	// 		The server will also send a ServiceIDs message when reconnecting.
	// 		That is, this method will also be executed when reconnecting.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#serviceids
	OnServiceIDs(message *protomsg.Message) error
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

	// DialTimeout sets the timeout value when connecting to the service.
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

	workerMng := &workerManager{
		workerMap: map[int32]*Worker{},
		rwMutex:   &sync.RWMutex{},
		bufSize:   channelBufSizePerStreamID,
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
		writeMutex:        &sync.Mutex{},
		workerMng:         workerMng,
	}

	return instance, nil
}

// awsClient is a structure that implements AWSClient interface.
type awsClient struct {
	mode                  Mode
	endpoint              *url.URL
	token                 string
	dialTimeout           time.Duration
	reconnectInterval     time.Duration
	pingInterval          time.Duration
	messageListeners      []AWSMessageListener
	connectHandlers       []func()
	dialer                *websocket.Dialer
	requestHeader         http.Header
	con                   *websocket.Conn
	writeMutex            *sync.Mutex             // for websocket.WriteMessage()
	workerMng             *workerManager          //
	unknownMessageHandler func(*protomsg.Message) // for unit testing
}

// Run Refer to AWSClient.
func (client *awsClient) Run(ctx context.Context) error {

	for {

		err := client.start(ctx)

		if err != nil {
			if conErr, ok := err.(*connectError); ok {
				// tunnel is closed
				if conErr.tunnelClosed() {
					return ErrTunnelClosed
				}

				// can not retry
				if !conErr.retryable() {
					return conErr
				}
			}

			// caller context is done
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}

			log.Warnf("Retryable connection error occur. %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(client.reconnectInterval):
		}

	}

}

func (client *awsClient) start(ctx context.Context) error {

	err := client.connect(ctx)
	if err != nil {
		return err
	}

	for _, connectHandler := range client.connectHandlers {
		connectHandler()
	}

	// TODO:need disconnect handler? defer for { disconnectHandler() }

	// -----------------------
	//  successful connection
	// -----------------------

	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start thread for sending ping.
	chSendPingTerminate := make(chan struct{}, 1)
	go func() {
		client.keepSendingPing(innerCtx)
		close(chSendPingTerminate)
	}()

	// start thread for reading messages.
	chReadMessageResult := make(chan error, 1)
	go func() {
		// When this thread terminates, it terminates the sending ping thread.
		defer cancel()
		chReadMessageResult <- client.keepReadingMessages(innerCtx)
	}()

	select {
	// caller context is done.
	case <-ctx.Done():
		// If connection is not closed, keepReadingMessages() will not be able to
		// exit the loop even if context is done.
		closeErr := client.close(websocket.CloseNormalClosure, "normal closure")
		if closeErr != nil {
			log.Error(closeErr)
		}

		// wait for reading thread to terminate.
		<-chReadMessageResult
		err = ctx.Err()
	// reading messages error.
	case err = <-chReadMessageResult:
		closeErr := client.close(websocket.CloseNormalClosure, "normal closure")
		if closeErr != nil {
			log.Error(closeErr)
		}
	}

	// wait for sending ping thread to terminate.
	<-chSendPingTerminate

	// stop all Worker
	client.workerMng.stopAll()

	return err
}

func (client *awsClient) connect(ctx context.Context) error {

	dialCtx, cancel := context.WithTimeout(ctx, client.dialTimeout)
	defer cancel()

	con, response, err := client.dialer.DialContext(
		dialCtx,
		client.endpoint.String(),
		client.requestHeader)

	if err != nil {
		err = &connectError{
			causeErr: err,
			url:      client.endpoint,
			response: response,
		}
		return err
	}

	log.Infof("Connected to secure tunneling service. url=%v", client.endpoint)

	if response != nil {
		log.Info("Response headers:")
		for key, value := range response.Header {
			log.Infof("  %s=%v", key, value)
		}
	}

	// WebSocket frames of up to 131076 bytes may be sent to clients
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#websocket-subprotocol-awsiotsecuretunneling-20
	con.SetReadLimit(maxWebSocketFrameSize)

	con.SetCloseHandler(func(code int, text string) error {
		log.Warnf("close websocket connection from server. code=%d error=%s", code, text)
		return nil
	})

	client.con = con

	return nil
}

// keepSendingPing keep sending ping frame.
// If failed to send ping frame, continue sending ping.
func (client *awsClient) keepSendingPing(ctx context.Context) {

	client.con.SetPongHandler(func(appData string) error {
		return nil
	})

	ticker := time.NewTicker(client.pingInterval)
	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():
			return

		case <-ticker.C:
			now := time.Now()
			message := now.Format("2006-01-02T15:04:05")
			timeout := now.Add(pingTimeout)

			err := client.con.WriteControl(websocket.PingMessage, []byte(message), timeout)
			if err != nil {
				log.Errorf("failed to send ping frame: %v", err)
				// continue sending ping
			}
		}

	}

}

// keepReadingMessages keep reading message frames,
// and fire event handlers associated with AWSClientOptions.MessageListeners.
func (client *awsClient) keepReadingMessages(ctx context.Context) error {

	for {

		select {

		case <-ctx.Done():
			return ctx.Err()

		default:

			messages, err := client.readMessages()
			if err != nil {
				return err
			}

			for _, message := range messages {
				// Perhaps Ignorable message can handle localproxy-specific data...?
				if message.Ignorable {
					continue
				}

				for _, handler := range client.messageListeners {
					client.invokeEvent(handler, message)
				}
			}

		}
	}
}

// readMessages read websocket frames, and deserialize.
func (client *awsClient) readMessages() ([]*protomsg.Message, error) {

	// A WebSocket frame may contain multiple tunneling frames,
	// **or it may contain only a slice of a tunneling frame started
	// in a previous WebSocket frame and will finish in a later WebSocket frame.**
	// This means that processing the WebSocket data must be done
	// as pure a sequence of bytes that sequentially construct tunneling frames
	// regardless of what the WebSocket fragmentation is.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#tunneling-message-frames

	// message -> secure tunnel message(protbuf message)
	// wsMessage -> websocket message

	prevMessageBin := []byte(nil)
	restMessageSize := uint16(0)
	messages := make([]*protomsg.Message, 0, 1)

	// loop to read websocket frame.
	for {

		wsMessageType, wsMessage, err := client.con.ReadMessage()
		if err != nil {
			return nil, err
		}

		// This protocol operates entirely with binary messages.
		// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#websocket-subprotocol-awsiotsecuretunneling-20
		if wsMessageType != websocket.BinaryMessage {
			return nil, errors.New("only binary messages can be accepted")
		}

		reader := bytes.NewReader(wsMessage)

		// loop to deserialize data -> to Message struct
		for {
			// |-----------------------------------------------------------------|
			// | 2-byte data length   |     N byte ProtocolBuffers message       |
			// |-----------------------------------------------------------------|
			// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#tunneling-message-frames

			var messageSize uint16

			if restMessageSize == 0 {
				// message size
				messageSizeBin := make([]byte, sizeOfMessageSize)
				_, err := reader.Read(messageSizeBin)
				if err != nil {
					err = fmt.Errorf("failed to read message size in websocket frame: %w", err)
					return nil, err
				}
				messageSize = binary.BigEndian.Uint16(messageSizeBin)
			} else {
				// Continuation from the previous websocket frame.
				messageSize = restMessageSize
			}

			// The entire binary of the message does not exist in this websocket frame.
			// The next message contains the remaining binaries.
			if reader.Len() < int(messageSize) {
				messageBin := make([]byte, reader.Len())
				readSize, err := reader.Read(messageBin)
				if err != nil {
					err = fmt.Errorf("failed to read partial message in websocket frame: %w", err)
					return nil, err
				}

				prevMessageBin = append(prevMessageBin, messageBin...)
				restMessageSize = messageSize - uint16(readSize)

				// next websocket.Conn.ReadMessage()
				break
			}

			// The last websocket frame for building a protobuf message.

			// binary protobuf message
			messageBin := make([]byte, messageSize)
			_, err = reader.Read(messageBin)
			if err != nil {
				err = fmt.Errorf("failed to read message in websocket frame: %w", err)
				return nil, err
			}

			if prevMessageBin != nil {
				messageBin = append(prevMessageBin, messageBin...)
			}

			prevMessageBin = nil
			restMessageSize = 0

			// deserialize
			message := &protomsg.Message{}
			err = proto.Unmarshal(messageBin, message)
			if err != nil {
				err = fmt.Errorf("invalid protobuf message format: %w", err)
				return nil, err
			}

			messages = append(messages, message)

			// EOF
			if reader.Len() == 0 {
				return messages, nil
			}
		}
	}
}

// invokeEvent invoke the appropriate event handler in AWSMessageListener according to the type of message.
func (client *awsClient) invokeEvent(
	messageListener AWSMessageListener,
	message *protomsg.Message) {

	switch message.Type {

	case protomsg.Message_STREAM_START:

		log.Infof(
			"received StreamStart message -> StreamID=StreamID=%d ServiceID=%s",
			message.StreamId,
			message.ServiceId)

		client.workerMng.start(message.StreamId)

		err := messageListener.OnStreamStart(message)
		if err != nil {
			log.Errorf(
				"OnStreamStart event failed -> StreamID=%d ServiceID=%s: %v",
				message.StreamId,
				message.ServiceId,
				err)

			err = client.SendStreamReset(message.StreamId, message.ServiceId)
			if err != nil {
				log.Error(err)
			}
		}

	case protomsg.Message_DATA:

		log.Debugf("received Data message StreamID=%d ServiceID=%s", message.StreamId, message.ServiceId)

		executed := client.workerMng.exec(message.StreamId, func(context.Context) {

			err := messageListener.OnData(message)
			if err != nil {
				log.Errorf(
					"OnData event failed -> StreamID=%d ServiceID=%s: %v",
					message.StreamId,
					message.ServiceId,
					err)

				err = client.SendStreamReset(message.StreamId, message.ServiceId)
				if err != nil {
					log.Error(err)
				}
			}
		})

		if !executed {
			log.Warnf("the StreamID has already been reset. StreamID=%d", message.StreamId)
		}

	case protomsg.Message_STREAM_RESET:

		log.Warnf(
			"received StreamReset message -> StreamID=StreamID=%d ServiceID=%s",
			message.StreamId,
			message.ServiceId)

		executed := client.workerMng.exec(message.StreamId, func(context.Context) {
			messageListener.OnStreamReset(message)
			client.workerMng.stop(message.StreamId)
		})

		if !executed {
			log.Warnf("the StreamID has already been reset. StreamID=%d", message.StreamId)
		}

	case protomsg.Message_SESSION_RESET:

		log.Warn("Received SessionReset message")
		messageListener.OnSessionReset(message)

	case protomsg.Message_SERVICE_IDS:

		log.Infof("Received ServiceIDs message ServiceID=%s", message.AvailableServiceIds)

		err := messageListener.OnServiceIDs(message)
		if err != nil {
			log.Errorf("OnServiceIDs() event failed: %v", err)
		}

	case protomsg.Message_UNKNOWN:
		fallthrough
	default:
		log.Errorf(
			"Invalid message was received -> StreamID=%d MessageType=%d Payload=%v",
			message.StreamId,
			message.Type,
			message.Payload)

		// for unit testing
		if client.unknownMessageHandler != nil {
			client.unknownMessageHandler(message)
		}
	}

}

// SendStreamStart Refer to AWSClient.
func (client *awsClient) SendStreamStart(streamID int32, serviceID string) error {

	log.Infof("Send StreamStart message StreamID=%d ServiceID=%s", streamID, serviceID)

	client.workerMng.start(streamID)

	err := client.sendMessage(streamID, serviceID, protomsg.Message_STREAM_START, nil)
	if err != nil {
		client.workerMng.stop(streamID)
		err = fmt.Errorf("failed to send StreamStart message: %w", err)
		return err
	}

	return nil
}

// SendStreamReset Refer to AWSClient.
func (client *awsClient) SendStreamReset(streamID int32, serviceID string) error {

	log.Warnf("Send StreamReset message StreamID=%d ServiceID=%s", streamID, serviceID)

	client.workerMng.stop(streamID)

	err := client.sendMessage(streamID, serviceID, protomsg.Message_STREAM_RESET, nil)
	if err != nil {
		err = fmt.Errorf("failed to send StreamReset message: %w", err)
		return err
	}

	return nil
}

// SendData Refer to AWSClient.
func (client *awsClient) SendData(streamID int32, serviceID string, data []byte) error {

	log.Debugf("SendData StreamID=%d", streamID)

	err := client.sendMessage(streamID, serviceID, protomsg.Message_DATA, data)
	if err != nil {
		err = fmt.Errorf("failed to send Data message: %w", err)
		return err
	}

	return nil
}

// sendMessage send secure tunneling message.
func (client *awsClient) sendMessage(
	streamID int32,
	serviceID string,
	messageType protomsg.Message_Type,
	data []byte) error {

	client.writeMutex.Lock()
	defer client.writeMutex.Unlock()

	// TODO: Should large messages be split and sent?

	message := &protomsg.Message{
		StreamId:  streamID,
		ServiceId: serviceID,
		Type:      messageType,
		Payload:   data,
	}

	// serialize message
	messageBin, err := proto.Marshal(message)
	if err != nil {
		err = fmt.Errorf("failed to serialize message: %w", err)
		return err
	}

	// size
	sizeBin := make([]byte, sizeOfMessageSize)
	binary.BigEndian.PutUint16(sizeBin, uint16(len(messageBin)))

	// format message size+message
	messageBin = append(sizeBin, messageBin...)

	// send
	err = client.con.WriteMessage(websocket.BinaryMessage, messageBin)
	if err != nil {
		err = fmt.Errorf("failed to send websocket message: %w", err)
		return err
	}

	return nil
}

// close send websocket close message and disconnect from server.
func (client *awsClient) close(closeCode int, text string) error {

	message := websocket.FormatCloseMessage(closeCode, text)
	timeout := time.Now().Add(time.Second)
	err := client.con.WriteControl(websocket.CloseMessage, message, timeout)
	if err != nil {
		err = fmt.Errorf("failed to close connection: %w", err)
		return err
	}

	err = client.con.Close()
	if err != nil {
		err = fmt.Errorf("failed to close connection: %w", err)
		return err
	}

	return nil
}

// connectError is a structure that represents connection error to endpoint.
type connectError struct {
	causeErr error
	url      *url.URL
	response *http.Response
}

// Error is an implementation of the error interface.
func (conErr *connectError) Error() string {

	var responseHeader http.Header
	if conErr.response != nil {
		responseHeader = conErr.response.Header
	}

	message := fmt.Sprintf(
		"failed to connect to ->%v header=%v cause=%v",
		conErr.url,
		responseHeader,
		conErr.causeErr)

	return message
}

// tunnelClosed returns whether the tunnel is closed.
func (conErr *connectError) tunnelClosed() bool {

	if conErr.response != nil {
		status := conErr.response.Header.Get(headerKeyStatusReason)
		result := (status == statusReasonTunnelClosed)
		return result
	}

	return false
}

// retryable returns whether it is an error that can be reconected.
// 	- response status 400 - 499
// 	- tunnel is closed
// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#handshake-error-responses
func (conErr *connectError) retryable() bool {

	if conErr.response != nil {
		statusCode := conErr.response.StatusCode

		if 400 <= statusCode && statusCode < 500 {
			return false
		}
	}

	return true
}
