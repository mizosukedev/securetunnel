package testutil

type OnReadDataArgs struct {
	StreamID  int32
	ServiceID string
	Data      []byte
}

type OnReadErrorArgs struct {
	StreamID  int32
	ServiceID string
	Err       error
}

type MockSocketReader struct {
	ChanOnReadDataArgs chan OnReadDataArgs
	MockOnReadData     func(streamID int32, serviceID string, data []byte) error
	ChanOnReadError    chan OnReadErrorArgs
	MockOnReadError    func(streamID int32, serviceID string, err error)
}

func NewMockSocketReader() *MockSocketReader {

	instance := &MockSocketReader{
		ChanOnReadDataArgs: make(chan OnReadDataArgs, 10),
		MockOnReadData: func(streamID int32, serviceID string, data []byte) error {
			return nil
		},
		ChanOnReadError: make(chan OnReadErrorArgs, 10),
		MockOnReadError: func(streamID int32, serviceID string, err error) {
		},
	}

	return instance
}

func (mock *MockSocketReader) OnReadData(streamID int32, serviceID string, data []byte) error {

	mock.ChanOnReadDataArgs <- OnReadDataArgs{streamID, serviceID, data}

	return mock.MockOnReadData(streamID, serviceID, data)
}

func (mock *MockSocketReader) OnReadError(streamID int32, serviceID string, err error) {
	mock.ChanOnReadError <- OnReadErrorArgs{streamID, serviceID, err}

	mock.MockOnReadError(streamID, serviceID, err)
}
