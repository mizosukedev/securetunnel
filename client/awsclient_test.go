package client

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/mizosukedev/securetunnel/_testutil"
	"github.com/mizosukedev/securetunnel/aws"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

type AWSClientTest struct {
	suite.Suite
}

func TestAWSClient(t *testing.T) {
	suite.Run(t, new(AWSClientTest))
}

// TextConnect confirm the request header when the AWSClient connects.
func (suite *AWSClientTest) TestConnect() {

	server := testutil.NewSecureTunnelServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Start(ctx)

	chConnected := make(chan struct{}, 1)

	messageListener := NewMockAWSMessageListener()

	options := defaultOptions()
	options.Endpoint = server.Endpoint
	options.MessageListeners = []AWSMessageListener{messageListener}
	options.ConnectHandlers = []func(){
		func() {
			chConnected <- struct{}{}
		},
	}

	// --------------------------
	//  Connect success

	client, err := NewAWSClient(options)
	suite.Require().Nil(err)
	clientCtx, clientCancel := context.WithCancel(context.Background())
	defer clientCancel()
	chClientTerminate := make(chan struct{}, 1)

	go func() {
		err := client.Run(clientCtx)
		suite.Require().Nil(err)
		chClientTerminate <- struct{}{}
	}()

	// wait for connection
	<-chConnected

	request := <-server.ChRequest

	// check request header
	suite.Require().Equal(string(aws.ModeDestination), request.FormValue(aws.QueryKeyProxyMode))
	suite.Require().Equal([]string{options.Token}, request.Header["Access-Token"])
	suite.Require().Equal(aws.SubProtocols, request.Header["Sec-Websocket-Protocol"])

	// wait for ping
	<-server.ChPing

	clientCancel()
	<-chClientTerminate
}

// TestReconnect confirm that AWS Client reconnects properly.
func (suite *AWSClientTest) TestReconnect() {

	type response struct {
		statusCode   int
		tunnelClosed bool
	}

	tests := []struct {
		name       string
		response   response
		wantRetry  bool
		wantRunErr bool
	}{
		{"status 399", response{399, false}, true, false},
		{"status 400", response{400, false}, false, true},
		{"tunnel closed", response{403, true}, false, true},
		{"status 499", response{499, false}, false, true},
		{"status 500", response{500, false}, true, false},
	}

	for _, test := range tests {

		server := testutil.NewSecureTunnelServer()
		ctx, cancel := context.WithCancel(context.Background())

		server.Start(ctx)

		messageListener := NewMockAWSMessageListener()

		options := defaultOptions()
		options.Endpoint = server.Endpoint
		options.MessageListeners = []AWSMessageListener{messageListener}

		server.RequestHandler = func(w http.ResponseWriter, r *http.Request) bool {
			if test.response.tunnelClosed {
				w.Header().Set(aws.HeaderKeyStatusReason, aws.StatusReasonTunnelClosed)
			}
			w.WriteHeader(test.response.statusCode)
			return true
		}

		client, err := NewAWSClient(options)
		suite.Require().Nil(err)

		chClientTerminate := make(chan struct{}, 1)

		go func() {
			err := client.Run(ctx)

			if test.wantRunErr {
				suite.Require().NotNil(err)
				fmt.Println(err)
			} else {
				suite.Require().Nil(err)
			}

			chClientTerminate <- struct{}{}
		}()

		if test.wantRetry {
			<-server.ChRequest // first connection
			<-server.ChRequest // retry
			cancel()
			<-chClientTerminate
		} else {
			<-chClientTerminate
		}

		cancel()
	}
}

// TestReceivedMessage confirm that AWSClient execute AWSMessageListener's event handler,
// when it receives messages.
func (suite *AWSClientTest) TestReceivedMessage() {

	server := testutil.NewSecureTunnelServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Start(ctx)

	chConnected := make(chan struct{}, 1)

	messageListener := NewMockAWSMessageListener()

	options := defaultOptions()
	options.Endpoint = server.Endpoint
	options.MessageListeners = []AWSMessageListener{messageListener}
	options.ConnectHandlers = []func(){
		func() {
			chConnected <- struct{}{}
		},
	}

	client, err := NewAWSClient(options)
	suite.Require().Nil(err)

	go func() {
		_ = client.Run(ctx)
	}()

	// wait for connection
	<-chConnected

	ws := <-server.ChWebSocket

	// wait for ping
	<-server.ChPing

	// -----------------------------
	//  receive StreamStart message
	streamStartMessage := &aws.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      aws.Message_STREAM_START,
	}
	messagesBin, err := marshalMessages([]*aws.Message{streamStartMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	receivedStreamStartMessage := <-messageListener.ChStreamStartArg
	suite.Require().Equal(streamStartMessage.StreamId, receivedStreamStartMessage.StreamId)
	suite.Require().Equal(streamStartMessage.ServiceId, receivedStreamStartMessage.ServiceId)

	// -----------------------------
	//  received a message divided into multiple websocket frames.
	//  receive Data message
	dataMessage := &aws.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      aws.Message_DATA,
		Payload:   []byte("01234567890123456789012345678901234567890123456789"),
	}

	messagesBin, err = marshalMessages([]*aws.Message{dataMessage})
	suite.Require().Nil(err)

	// divide message
	halfSize := int(len(messagesBin) / 2)
	firstMessageBin := messagesBin[:halfSize]
	secondMessageBin := messagesBin[halfSize:]

	err = ws.WriteMessage(websocket.BinaryMessage, firstMessageBin)
	suite.Require().Nil(err)
	err = ws.WriteMessage(websocket.BinaryMessage, secondMessageBin)
	suite.Require().Nil(err)

	receivedDataMessage := <-messageListener.ChDataArg
	suite.Require().Equal(dataMessage.StreamId, receivedDataMessage.StreamId)
	suite.Require().Equal(dataMessage.ServiceId, receivedDataMessage.ServiceId)
	suite.Require().Equal(dataMessage.Payload, receivedDataMessage.Payload)

	// -----------------------------
	//  receive StreamReset message
	streamResetMessage := &aws.Message{
		StreamId:  1,
		ServiceId: "serviceID2",
		Type:      aws.Message_STREAM_RESET,
	}
	messagesBin, err = marshalMessages([]*aws.Message{streamResetMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	receivedStreamResetMessage := <-messageListener.ChStreamResetArg
	suite.Require().Equal(streamResetMessage.StreamId, receivedStreamResetMessage.StreamId)
	suite.Require().Equal(streamResetMessage.ServiceId, receivedStreamResetMessage.ServiceId)

	// confirm that Worker was stopped
	stopped := false
	for i := 0; i < 10; i++ {

		workers := client.(*awsClient).workerMng.getAll()
		if len(workers) == 0 {
			stopped = true
			break
		}

		time.Sleep(time.Millisecond * 100)
	}

	suite.Require().True(stopped)

	// -----------------------------
	//  receive multiple messages in a websocket frame
	//  receive SessionReset message
	//  receive ServiceIDs message
	sessionResetMessage := &aws.Message{
		Type: aws.Message_SESSION_RESET,
	}

	serviceIDsMessage := &aws.Message{
		AvailableServiceIds: []string{"service1", "service2"},
		Type:                aws.Message_SERVICE_IDS,
	}

	messages := []*aws.Message{
		sessionResetMessage,
		serviceIDsMessage,
	}

	messagesBin, err = marshalMessages(messages)
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	<-messageListener.ChSessionResetArg
	receivedServiceIDsMessage := <-messageListener.ChServiceIDsArg
	suite.Require().Equal(receivedServiceIDsMessage.AvailableServiceIds, serviceIDsMessage.AvailableServiceIds)
}

// TestReceivedMessageListenerReturnsError confirm the behavior
// when AWSMessageListener's event handlers returns error.
func (suite *AWSClientTest) TestReceivedMessageListenerReturnsError() {

	server := testutil.NewSecureTunnelServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Start(ctx)

	chConnected := make(chan struct{}, 1)

	messageListener := NewMockAWSMessageListener()

	options := defaultOptions()
	options.Endpoint = server.Endpoint
	options.MessageListeners = []AWSMessageListener{messageListener}
	options.ConnectHandlers = []func(){
		func() {
			chConnected <- struct{}{}
		},
	}

	client, err := NewAWSClient(options)
	suite.Require().Nil(err)

	go func() {
		_ = client.Run(ctx)
	}()

	// wait for connection
	<-chConnected

	ws := <-server.ChWebSocket
	request := <-server.ChRequest

	// check request header
	suite.Require().Equal(string(aws.ModeDestination), request.FormValue(aws.QueryKeyProxyMode))
	suite.Require().Equal([]string{options.Token}, request.Header["Access-Token"])
	suite.Require().Equal(aws.SubProtocols, request.Header["Sec-Websocket-Protocol"])

	// -----------------------------
	//  StreamStart message
	messageListener.MockOnStreamStart = func(message *aws.Message) error {
		return errors.New("test error")
	}

	streamStartMessage := &aws.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      aws.Message_STREAM_START,
	}
	messagesBin, err := marshalMessages([]*aws.Message{streamStartMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	<-messageListener.ChStreamStartArg

	// confirm that AWSClient send StreamReset message.
	readMessageResult := <-server.ChMessage
	streamResetMessage, err := readMessageResult.UnmarshalMessage()
	suite.Require().Nil(err)
	suite.Require().Equal(aws.Message_STREAM_RESET, streamResetMessage.Type)
	suite.Require().Equal(streamStartMessage.StreamId, streamResetMessage.StreamId)

	messageListener.MockOnStreamStart = func(message *aws.Message) error {
		return nil
	}

	// -----------------------------
	//  Data message
	messageListener.MockOnData = func(message *aws.Message) error {
		return errors.New("test error")
	}

	dataMessage := &aws.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      aws.Message_DATA,
		Payload:   []byte{},
	}

	messagesBin, err = marshalMessages([]*aws.Message{streamStartMessage, dataMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	// confirm that AWSClient send StreamReset message.
	readMessageResult = <-server.ChMessage
	dataMessage, err = readMessageResult.UnmarshalMessage()
	suite.Require().Nil(err)
	suite.Require().Equal(aws.Message_STREAM_RESET, dataMessage.Type)
	suite.Require().Equal(streamStartMessage.StreamId, dataMessage.StreamId)

	// confirm that Worker was stopped
	stopped := false
	for i := 0; i < 10; i++ {

		workers := client.(*awsClient).workerMng.getAll()
		if len(workers) == 0 {
			stopped = true
			break
		}

		time.Sleep(time.Millisecond * 100)
	}

	suite.Require().True(stopped)

	messageListener.MockOnData = func(message *aws.Message) error {
		return nil
	}

	// -----------------------------
	//  ServiceIDs message
	messageListener.MockServiceIDs = func(message *aws.Message) error {
		return errors.New("test error")
	}

	serviceIDsMessage := &aws.Message{
		Type:                aws.Message_DATA,
		AvailableServiceIds: []string{},
	}

	messagesBin, err = marshalMessages([]*aws.Message{serviceIDsMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	messageListener.MockServiceIDs = func(message *aws.Message) error {
		return nil
	}
}

// TestReceivedMessageListenerReturnsError confirm the behavior
// when AWSClient received invalid messages.
func (suite *AWSClientTest) TestReceivedInvalidMessage() {

	server := testutil.NewSecureTunnelServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Start(ctx)

	chConnected := make(chan struct{}, 1)

	messageListener := NewMockAWSMessageListener()

	options := defaultOptions()
	options.Endpoint = server.Endpoint
	options.MessageListeners = []AWSMessageListener{messageListener}
	options.ConnectHandlers = []func(){
		func() {
			chConnected <- struct{}{}
		},
	}

	client, err := NewAWSClient(options)
	suite.Require().Nil(err)

	go func() {
		_ = client.Run(ctx)
	}()

	// wait for connection
	<-chConnected
	ws := <-server.ChWebSocket

	// -----------------------------
	//  text message
	sessionResetMessage := &aws.Message{
		Type: aws.Message_SESSION_RESET,
	}
	messagesBin, err := marshalMessages([]*aws.Message{sessionResetMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.TextMessage, messagesBin)
	suite.Require().Nil(err)

	// confirm reconnection
	<-chConnected
	ws = <-server.ChWebSocket

	// -----------------------------
	//  receive unknown message
	chUnknownMessage := make(chan struct{}, 1)
	client.(*awsClient).unknownMessageHandler = func(message *aws.Message) {
		close(chUnknownMessage)
	}

	unknownMessage := &aws.Message{
		Type:    aws.Message_UNKNOWN,
		Payload: []byte("a"),
	}

	messagesBin, err = marshalMessages([]*aws.Message{unknownMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	<-chUnknownMessage

	client.(*awsClient).unknownMessageHandler = func(message *aws.Message) {}

	// -----------------------------
	//  ignorable message
	ignorableMessage := &aws.Message{
		StreamId:  1,
		ServiceId: "service1",
		Type:      aws.Message_STREAM_START,
		Ignorable: true,
	}

	messagesBin, err = marshalMessages([]*aws.Message{ignorableMessage, sessionResetMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	<-messageListener.ChSessionResetArg

	suite.Require().Len(messageListener.ChStreamStartArg, 0)
}

func (suite *AWSClientTest) TestSendMessage() {

	server := testutil.NewSecureTunnelServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Start(ctx)

	chConnected := make(chan struct{}, 1)

	messageListener := NewMockAWSMessageListener()

	options := defaultOptions()
	options.Endpoint = server.Endpoint
	options.MessageListeners = []AWSMessageListener{messageListener}
	options.ConnectHandlers = []func(){
		func() {
			chConnected <- struct{}{}
		},
	}

	client, err := NewAWSClient(options)
	suite.Require().Nil(err)

	go func() {
		_ = client.Run(ctx)
	}()

	// wait for connection
	<-chConnected

	streamID := int32(1)
	serviceID := "service 1"
	data := []byte("test test test")

	// ---------------------------
	//  semd StreamStart
	err = client.SendStreamStart(streamID, serviceID)
	suite.Require().Nil(err)

	readMessageResult := <-server.ChMessage
	message, err := readMessageResult.UnmarshalMessage()
	suite.Require().Nil(err)

	suite.Require().Equal(streamID, message.StreamId)
	suite.Require().Equal(serviceID, message.ServiceId)

	// ---------------------------
	//  semd Data
	err = client.SendData(streamID, serviceID, data)
	suite.Require().Nil(err)

	readMessageResult = <-server.ChMessage
	message, err = readMessageResult.UnmarshalMessage()
	suite.Require().Nil(err)

	suite.Require().Equal(streamID, message.StreamId)
	suite.Require().Equal(serviceID, message.ServiceId)
	suite.Require().Equal(data, message.Payload)

	// ---------------------------
	//  semd StreamReset
	err = client.SendStreamReset(streamID, serviceID)
	suite.Require().Nil(err)

	readMessageResult = <-server.ChMessage
	message, err = readMessageResult.UnmarshalMessage()
	suite.Require().Nil(err)

	suite.Require().Equal(streamID, message.StreamId)
	suite.Require().Equal(serviceID, message.ServiceId)
}

func defaultOptions() AWSClientOptions {

	options := AWSClientOptions{
		Mode:              aws.ModeDestination,
		Token:             "test_token_string",
		Endpoint:          nil,
		TLSConfig:         nil,
		DialTimeout:       time.Second * 5,
		ReconnectInterval: time.Millisecond * 10,
		PingInterval:      time.Millisecond * 100,
		MessageListeners:  []AWSMessageListener{},
		ConnectHandlers:   []func(){},
	}
	return options
}

func marshalMessages(messages []*aws.Message) ([]byte, error) {

	messagesBin := make([]byte, 0)

	for _, message := range messages {

		messageBin, err := proto.Marshal(message)
		if err != nil {
			return nil, err
		}

		sizeBin := make([]byte, aws.SizeOfMessageSize)
		binary.BigEndian.PutUint16(sizeBin, uint16(len(messageBin)))

		messagesBin = append(messagesBin, sizeBin...)
		messagesBin = append(messagesBin, messageBin...)
	}

	return messagesBin, nil
}
