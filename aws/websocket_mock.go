package aws

type MockWebSocketReader struct {
	MockRead func() (messageType int, p []byte, err error)
}

func NewMockWebSocketReader() *MockWebSocketReader {

	instance := &MockWebSocketReader{}

	instance.MockRead = func() (messageType int, p []byte, err error) {
		return 0, nil, nil
	}

	return instance
}

func (mock *MockWebSocketReader) ReadMessage() (messageType int, p []byte, err error) {
	return mock.MockRead()
}
