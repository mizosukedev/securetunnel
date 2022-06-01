package aws

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// MessageReader is an interface for reading secure tunneling messages.
type MessageReader interface {
	// Read websocket frames, and return deserialized Message.
	Read() (*Message, error)
}

// WebSocketReader is an interface for reading websocket frames.
// Actually, it refers to gorilla/websocket.Conn.ReadMessage().
type WebSocketReader interface {
	ReadMessage() (messageType int, p []byte, err error)
}

// NewMessageReader returns a instance implements MessageReader interface.
func NewMessageReader(con WebSocketReader) MessageReader {

	instance := &messageReaderImpl{
		con:         con,
		messageSize: 0,
		messageBin:  nil,
	}

	return instance
}

type messageReaderImpl struct {
	con         WebSocketReader
	messageSize uint16
	messageBin  []byte
}

// Read websocket frames, and return deserialized Message.
func (reader *messageReaderImpl) Read() (*Message, error) {

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

	err := reader.read(SizeOfMessageSize)
	if err != nil {
		return nil, err
	}

	// read message size
	sizeBin := reader.messageBin[:SizeOfMessageSize]
	reader.messageSize = binary.BigEndian.Uint16(sizeBin)

	endOfMessagePosition := SizeOfMessageSize + reader.messageSize

	// Continue to execute ReadMessage() until the data that deserializes one protobuf message is collected.
	err = reader.read(endOfMessagePosition)
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

// read data of `leastSize` bytes or more.
func (reader *messageReaderImpl) read(leastSize uint16) error {

	for len(reader.messageBin) < int(leastSize) {

		wsMessageType, wsMessage, err := reader.con.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read websocket message: %w", err)
		}

		// This protocol operates entirely with binary messages.
		// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#websocket-subprotocol-awsiotsecuretunneling-20
		if wsMessageType != websocket.BinaryMessage {
			return errors.New("only binary messages can be accepted")
		}

		reader.messageBin = append(reader.messageBin, wsMessage...)
	}

	return nil
}

// func (reader *messageReaderImpl) shouldReadMessage() bool {

// 	if reader.messageSize == 0 {
// 		return true
// 	} else {
// 		// Not enough data to read.
// 		if int(reader.messageSize) < len(reader.messageBin)-SizeOfMessageSize {
// 			return true
// 		} else {
// 			return false
// 		}
// 	}
// }

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
