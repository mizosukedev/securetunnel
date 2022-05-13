package client

import "github.com/mizosukedev/securetunnel/protomsg"

type MockAWSMessageListener struct {
	ChStreamStartArg   chan *protomsg.Message
	MockOnStreamStart  func(message *protomsg.Message) error
	ChStreamResetArg   chan *protomsg.Message
	MockOnStreamReset  func(message *protomsg.Message)
	ChSessionResetArg  chan *protomsg.Message
	MockOnSessionReset func(message *protomsg.Message)
	ChDataArg          chan *protomsg.Message
	MockOnData         func(message *protomsg.Message) error
	ChServiceIDsArg    chan *protomsg.Message
	MockServiceIDs     func(message *protomsg.Message) error
}

func NewMockAWSMessageListener() *MockAWSMessageListener {

	instance := &MockAWSMessageListener{
		ChStreamStartArg: make(chan *protomsg.Message, 10),
		MockOnStreamStart: func(*protomsg.Message) error {
			return nil
		},
		ChStreamResetArg:   make(chan *protomsg.Message, 10),
		MockOnStreamReset:  func(*protomsg.Message) {},
		ChSessionResetArg:  make(chan *protomsg.Message, 10),
		MockOnSessionReset: func(*protomsg.Message) {},
		ChDataArg:          make(chan *protomsg.Message, 10),
		MockOnData: func(*protomsg.Message) error {
			return nil
		},
		ChServiceIDsArg: make(chan *protomsg.Message, 10),
		MockServiceIDs: func(*protomsg.Message) error {
			return nil
		},
	}

	return instance
}

func (mock *MockAWSMessageListener) OnStreamStart(message *protomsg.Message) error {
	mock.ChStreamStartArg <- message
	return mock.MockOnStreamStart(message)
}

func (mock *MockAWSMessageListener) OnStreamReset(message *protomsg.Message) {
	mock.ChStreamResetArg <- message
	mock.MockOnStreamReset(message)
}

func (mock *MockAWSMessageListener) OnSessionReset(message *protomsg.Message) {
	mock.ChSessionResetArg <- message
	mock.MockOnSessionReset(message)
}

func (mock *MockAWSMessageListener) OnData(message *protomsg.Message) error {
	mock.ChDataArg <- message
	return mock.MockOnData(message)
}

func (mock *MockAWSMessageListener) OnServiceIDs(message *protomsg.Message) error {
	mock.ChServiceIDsArg <- message
	return mock.MockServiceIDs(message)
}
