package aws

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// MessageReader is an interface for reading messages in secure tunneling service.
type MessageReader interface {
	// Next reads websocket frames, and returns next deserialized Message.
	Next() (*Message, error)
}

// BinaryReader is an interface for reading binary data.
type BinaryReader interface {
	Read() ([]byte, error)
}

// WebSocketReader is an interface for reading websocket frames.
// Actually, it refers to gorilla/websocket.Conn.ReadMessage().
type WebSocketReader interface {
	ReadMessage() (messageType int, p []byte, err error)
}

// NewBReaderFromWSReader returns BinaryReader from WebSocketReader.
func NewBReaderFromWSReader(wsReader WebSocketReader) BinaryReader {

	instance := &binaryReaderWSAdapter{
		wsReader: wsReader,
	}

	return instance
}

// binaryReaderWSAdapter is an adapter for converting gorilla/websocket.Conn to BinaryReader.
type binaryReaderWSAdapter struct {
	wsReader WebSocketReader
}

// Read message from websocket.Conn and return it.
func (reader *binaryReaderWSAdapter) Read() ([]byte, error) {

	wsMessageType, wsMessage, err := reader.wsReader.ReadMessage()

	// if ReadMessage() returns an error, the value of wsMessageType is 0.
	// So check for error first.
	if err != nil {
		return nil, err
	}

	// This protocol operates entirely with binary messages.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#websocket-subprotocol-awsiotsecuretunneling-20
	if wsMessageType != websocket.BinaryMessage {
		return nil, errors.New("only binary messages can be accepted")
	}

	return wsMessage, nil
}

// NewMessageReader returns a instance implements MessageReader interface.
func NewMessageReader(binReader BinaryReader) MessageReader {

	instance := &messageReaderImpl{
		binReader:   binReader,
		messageSize: 0,
		messageBin:  nil,
	}

	return instance
}

type messageReaderImpl struct {
	binReader   BinaryReader
	messageSize uint16
	messageBin  []byte
}

// Next reads websocket frames, and returns next deserialized Message.
// If the data read from the BinaryReader remains after returning the Message,
// that data will be stored internally until the next method call.
func (reader *messageReaderImpl) Next() (*Message, error) {

	// A WebSocket frame may contain multiple tunneling frames,
	// **or it may contain only a slice of a tunneling frame started
	// in a previous WebSocket frame and will finish in a later WebSocket frame.**
	// This means that processing the WebSocket data must be done
	// as pure a sequence of bytes that sequentially construct tunneling frames
	// regardless of what the WebSocket fragmentation is.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#tunneling-message-frames

	// |-----------------------------------------------------------------|
	// | 2-byte data length   |     N byte ProtocolBuffers message       |
	// |-----------------------------------------------------------------|
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#tunneling-message-frames

	err := reader.readAtLeast(SizeOfMessageSize)
	if err != nil {
		return nil, err
	}

	// read message size
	sizeBin := reader.messageBin[:SizeOfMessageSize]
	reader.messageSize = binary.BigEndian.Uint16(sizeBin)

	endOfMessagePosition := SizeOfMessageSize + reader.messageSize

	// Continue to execute ReadMessage() until the data that deserializes one protobuf message is collected.
	err = reader.readAtLeast(endOfMessagePosition)
	if err != nil {
		return nil, err
	}

	// read binary protobuf message body
	messageBin := reader.messageBin[SizeOfMessageSize:endOfMessagePosition]

	reader.messageSize = 0
	reader.messageBin = reader.messageBin[endOfMessagePosition:]

	// deserialize
	message := &Message{}
	err = proto.Unmarshal(messageBin, message)
	if err != nil {
		err = fmt.Errorf("invalid protobuf message format: %w", err)
		return nil, err
	}

	return message, nil
}

// readAtLeast read data at least `leastSize` bytes or more.
func (reader *messageReaderImpl) readAtLeast(leastSize uint16) error {

	for len(reader.messageBin) < int(leastSize) {

		binMessage, err := reader.binReader.Read()
		if err != nil {
			return fmt.Errorf("failed to read websocket message: %w", err)
		}

		reader.messageBin = append(reader.messageBin, binMessage...)
	}

	return nil
}

// SerializeMessage returns serialized binary data.
func SerializeMessage(message *Message) ([]byte, error) {

	// serialize message
	messageBin, err := proto.Marshal(message)
	if err != nil {
		err = fmt.Errorf("failed to serialize message: %w", err)
		return nil, err
	}

	// add size
	sizeBin := make([]byte, SizeOfMessageSize)
	binary.BigEndian.PutUint16(sizeBin, uint16(len(messageBin)))

	// format message size+message
	messageBin = append(sizeBin, messageBin...)

	return messageBin, nil
}
