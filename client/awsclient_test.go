package client

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/mizosukedev/securetunnel/_testutil"
	"github.com/mizosukedev/securetunnel/protomsg"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

type AWSClientTest struct {
	suite.Suite
}

func TestAWSClient(t *testing.T) {
	suite.Run(t, new(AWSClientTest))
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

	go client.Run(ctx)

	// wait for connection
	<-chConnected

	ws := <-server.ChWebSocket
	request := <-server.ChRequest

	// check request header
	suite.Require().Equal(string(ModeDestination), request.FormValue(queryKeyProxyMode))
	suite.Require().Equal([]string{options.Token}, request.Header["Access-Token"])
	suite.Require().Equal(subProtocols, request.Header["Sec-Websocket-Protocol"])

	// wait for ping
	<-server.ChPing

	// -----------------------------
	//  receive StreamStart message
	streamStartMessage := &protomsg.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      protomsg.Message_STREAM_START,
		Ignorable: false,
	}
	messagesBin, err := marshalMessage([]*protomsg.Message{streamStartMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	receivedStreamStartMessage := <-messageListener.ChStreamStartArg
	suite.Require().Equal(streamStartMessage.StreamId, receivedStreamStartMessage.StreamId)
	suite.Require().Equal(streamStartMessage.ServiceId, receivedStreamStartMessage.ServiceId)

	// -----------------------------
	//  received a message divided into multiple websocket frames.
	//  receive Data message
	dataMessage := &protomsg.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      protomsg.Message_DATA,
		Payload:   []byte("01234567890123456789012345678901234567890123456789"),
		Ignorable: false,
	}

	messagesBin, err = marshalMessage([]*protomsg.Message{dataMessage})
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
	streamResetMessage := &protomsg.Message{
		StreamId:  1,
		ServiceId: "serviceID2",
		Type:      protomsg.Message_STREAM_RESET,
		Ignorable: false,
	}
	messagesBin, err = marshalMessage([]*protomsg.Message{streamResetMessage})
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
	sessionResetMessage := &protomsg.Message{
		Type:      protomsg.Message_SESSION_RESET,
		Ignorable: false,
	}

	serviceIDsMessage := &protomsg.Message{
		AvailableServiceIds: []string{"service1", "service2"},
		Type:                protomsg.Message_SERVICE_IDS,
		Ignorable:           false,
	}

	messages := []*protomsg.Message{
		sessionResetMessage,
		serviceIDsMessage,
	}

	messagesBin, err = marshalMessage(messages)
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

	go client.Run(ctx)

	// wait for connection
	<-chConnected

	ws := <-server.ChWebSocket
	request := <-server.ChRequest

	// check request header
	suite.Require().Equal(string(ModeDestination), request.FormValue(queryKeyProxyMode))
	suite.Require().Equal([]string{options.Token}, request.Header["Access-Token"])
	suite.Require().Equal(subProtocols, request.Header["Sec-Websocket-Protocol"])

	// -----------------------------
	//  StreamStart message
	messageListener.MockOnStreamStart = func(message *protomsg.Message) error {
		return errors.New("test error")
	}

	streamStartMessage := &protomsg.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      protomsg.Message_STREAM_START,
		Ignorable: false,
	}
	messagesBin, err := marshalMessage([]*protomsg.Message{streamStartMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	<-messageListener.ChStreamStartArg

	readMessageResult := <-server.ChMessage
	streamResetMessage := &protomsg.Message{}
	err = proto.Unmarshal(readMessageResult.Message[sizeOfMessageSize:], streamResetMessage)
	suite.Require().Nil(err)
	suite.Require().Equal(protomsg.Message_STREAM_RESET, streamResetMessage.Type)
	suite.Require().Equal(streamStartMessage.StreamId, streamResetMessage.StreamId)

	messageListener.MockOnStreamStart = func(message *protomsg.Message) error {
		return nil
	}

	// -----------------------------
	//  Data message
	messageListener.MockOnData = func(message *protomsg.Message) error {
		return errors.New("test error")
	}

	dataMessage := &protomsg.Message{
		StreamId:  1,
		ServiceId: "serviceID1",
		Type:      protomsg.Message_DATA,
		Payload:   []byte{},
		Ignorable: false,
	}

	messagesBin, err = marshalMessage([]*protomsg.Message{streamStartMessage, dataMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	readMessageResult = <-server.ChMessage
	dataMessage = &protomsg.Message{}
	err = proto.Unmarshal(readMessageResult.Message[sizeOfMessageSize:], dataMessage)
	suite.Require().Nil(err)
	suite.Require().Equal(protomsg.Message_STREAM_RESET, dataMessage.Type)
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

	messageListener.MockOnData = func(message *protomsg.Message) error {
		return nil
	}

	// -----------------------------
	//  ServiceIDs message
	messageListener.MockServiceIDs = func(message *protomsg.Message) error {
		return errors.New("test error")
	}

	serviceIDsMessage := &protomsg.Message{
		Type:                protomsg.Message_DATA,
		AvailableServiceIds: []string{},
		Ignorable:           false,
	}

	messagesBin, err = marshalMessage([]*protomsg.Message{serviceIDsMessage})
	suite.Require().Nil(err)

	err = ws.WriteMessage(websocket.BinaryMessage, messagesBin)
	suite.Require().Nil(err)

	messageListener.MockServiceIDs = func(message *protomsg.Message) error {
		return nil
	}
}

// sendMessage
// reconnect
// tunnel closed
// ReadMessage error
// context.cancel

func defaultOptions() AWSClientOptions {

	options := AWSClientOptions{
		Mode:              ModeDestination,
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

func marshalMessage(messages []*protomsg.Message) ([]byte, error) {

	messagesBin := make([]byte, 0)

	for _, message := range messages {

		messageBin, err := proto.Marshal(message)
		if err != nil {
			return nil, err
		}

		sizeBin := make([]byte, sizeOfMessageSize)
		binary.BigEndian.PutUint16(sizeBin, uint16(len(messageBin)))

		messagesBin = append(messagesBin, sizeBin...)
		messagesBin = append(messagesBin, messageBin...)
	}

	return messagesBin, nil
}
