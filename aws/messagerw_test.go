package aws

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"
)

type MessageRWTest struct {
	suite.Suite
}

func TestMessageRW(t *testing.T) {
	suite.Run(t, new(MessageRWTest))
}

// TestRead confirm the operation when Worker is used normally.
func (suite *MessageRWTest) TestRead() {

	// create test data
	expectedMessage := &Message{
		Type:      Message_STREAM_START,
		StreamId:  int32(1),
		ServiceId: "serviceID",
		Payload:   []byte("abcdefghijklmn"),
	}

	messageBin, err := SerializeMessage(expectedMessage)
	suite.Require().Nil(err)

	websocketReader := NewMockWebSocketReader()
	websocketReader.MockRead = func() (messageType int, p []byte, err error) {
		return websocket.BinaryMessage, messageBin, nil
	}

	// test
	binReader := NewBReaderFromWSReader(websocketReader)
	reader := NewMessageReader(binReader)

	actualMessage, err := reader.Read()
	suite.Require().Nil(err)

	// assert
	suite.Require().Equal(expectedMessage.Type, actualMessage.Type)
	suite.Require().Equal(expectedMessage.StreamId, actualMessage.StreamId)
	suite.Require().Equal(expectedMessage.ServiceId, actualMessage.ServiceId)
	suite.Require().Equal(expectedMessage.Payload, actualMessage.Payload)
}

// TestReadMultipleMessagesInFrame multiple messages in a websocket frame
func (suite *MessageRWTest) TestReadMultipleMessagesInFrame() {

	// create data
	expectedMessages := []*Message{
		{
			Type:      Message_STREAM_START,
			StreamId:  int32(1),
			ServiceId: "serviceID",
			Payload:   []byte("abcdefghijklmn"),
		},
		{
			Type:                Message_SERVICE_IDS,
			StreamId:            int32(2),
			AvailableServiceIds: []string{"serviceID1", "serviceID2"},
		},
		{
			Type:      Message_DATA,
			StreamId:  int32(2),
			ServiceId: "serviceID2",
			Payload:   []byte("123456789"),
		},
	}

	var messagesBin []byte

	for _, message := range expectedMessages {
		messageBin, err := SerializeMessage(message)
		suite.Require().Nil(err)
		messagesBin = append(messagesBin, messageBin...)
	}

	websocketReader := NewMockWebSocketReader()
	websocketReader.MockRead = func() (messageType int, p []byte, err error) {
		return websocket.BinaryMessage, messagesBin, nil
	}

	// test
	binReader := NewBReaderFromWSReader(websocketReader)
	reader := NewMessageReader(binReader)

	for _, expectedMessage := range expectedMessages {

		actualMessage, err := reader.Read()
		suite.Require().Nil(err)

		// assert
		suite.Require().Equal(expectedMessage.Type, actualMessage.Type)
		suite.Require().Equal(expectedMessage.StreamId, actualMessage.StreamId)
		suite.Require().Equal(expectedMessage.ServiceId, actualMessage.ServiceId)
		suite.Require().Equal(expectedMessage.AvailableServiceIds, actualMessage.AvailableServiceIds)
		suite.Require().Equal(expectedMessage.Payload, actualMessage.Payload)
	}
}

// TestReadMessagesInMultipleFrame messages in multiple websocket frames.
func (suite *MessageRWTest) TestReadMessagesInMultipleFrame() {

	// create data
	expectedMessages := []*Message{
		{
			Type:      Message_STREAM_START,
			StreamId:  int32(1),
			ServiceId: "serviceID",
			Payload:   []byte("abcdefghijklmn"),
		},
		{
			Type:                Message_SERVICE_IDS,
			StreamId:            int32(2),
			AvailableServiceIds: []string{"serviceID1", "serviceID2"},
		},
	}

	var messagesBin []byte

	for _, message := range expectedMessages {
		messageBin, err := SerializeMessage(message)
		suite.Require().Nil(err)
		messagesBin = append(messagesBin, messageBin...)
	}

	// returns 1 byte at a time
	readIndex := 0
	websocketReader := NewMockWebSocketReader()
	websocketReader.MockRead = func() (messageType int, p []byte, err error) {

		bin := messagesBin[readIndex : readIndex+1]
		readIndex++

		return websocket.BinaryMessage, bin, nil
	}

	// test
	binReader := NewBReaderFromWSReader(websocketReader)
	reader := NewMessageReader(binReader)

	for _, expectedMessage := range expectedMessages {

		actualMessage, err := reader.Read()
		suite.Require().Nil(err)

		// assert
		suite.Require().Equal(expectedMessage.Type, actualMessage.Type)
		suite.Require().Equal(expectedMessage.StreamId, actualMessage.StreamId)
		suite.Require().Equal(expectedMessage.ServiceId, actualMessage.ServiceId)
		suite.Require().Equal(expectedMessage.AvailableServiceIds, actualMessage.AvailableServiceIds)
		suite.Require().Equal(expectedMessage.Payload, actualMessage.Payload)
	}
}

// TestReadTextWebsocketFrame recieve text websocket frame.
func (suite *MessageRWTest) TestReadTextWebsocketFrame() {

	// create test data
	message := &Message{
		Type:      Message_STREAM_START,
		StreamId:  int32(1),
		ServiceId: "serviceID",
		Payload:   []byte("abcdefghijklmn"),
	}

	messageBin, err := SerializeMessage(message)
	suite.Require().Nil(err)

	websocketReader := NewMockWebSocketReader()
	websocketReader.MockRead = func() (messageType int, p []byte, err error) {
		// text message
		return websocket.TextMessage, messageBin, nil
	}

	// test
	binReader := NewBReaderFromWSReader(websocketReader)
	reader := NewMessageReader(binReader)

	actualMessage, err := reader.Read()
	suite.Require().NotNil(err)
	suite.Require().Nil(actualMessage)
}
