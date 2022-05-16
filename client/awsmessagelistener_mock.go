package client

import "github.com/mizosukedev/securetunnel/aws"

type MockAWSMessageListener struct {
	ChStreamStartArg   chan *aws.Message
	MockOnStreamStart  func(message *aws.Message) error
	ChStreamResetArg   chan *aws.Message
	MockOnStreamReset  func(message *aws.Message)
	ChSessionResetArg  chan *aws.Message
	MockOnSessionReset func(message *aws.Message)
	ChDataArg          chan *aws.Message
	MockOnData         func(message *aws.Message) error
	ChServiceIDsArg    chan *aws.Message
	MockServiceIDs     func(message *aws.Message) error
}

func NewMockAWSMessageListener() *MockAWSMessageListener {

	instance := &MockAWSMessageListener{
		ChStreamStartArg: make(chan *aws.Message, 10),
		MockOnStreamStart: func(*aws.Message) error {
			return nil
		},
		ChStreamResetArg:   make(chan *aws.Message, 10),
		MockOnStreamReset:  func(*aws.Message) {},
		ChSessionResetArg:  make(chan *aws.Message, 10),
		MockOnSessionReset: func(*aws.Message) {},
		ChDataArg:          make(chan *aws.Message, 10),
		MockOnData: func(*aws.Message) error {
			return nil
		},
		ChServiceIDsArg: make(chan *aws.Message, 10),
		MockServiceIDs: func(*aws.Message) error {
			return nil
		},
	}

	return instance
}

func (mock *MockAWSMessageListener) OnStreamStart(message *aws.Message) error {
	mock.ChStreamStartArg <- message
	return mock.MockOnStreamStart(message)
}

func (mock *MockAWSMessageListener) OnStreamReset(message *aws.Message) {
	mock.ChStreamResetArg <- message
	mock.MockOnStreamReset(message)
}

func (mock *MockAWSMessageListener) OnSessionReset(message *aws.Message) {
	mock.ChSessionResetArg <- message
	mock.MockOnSessionReset(message)
}

func (mock *MockAWSMessageListener) OnData(message *aws.Message) error {
	mock.ChDataArg <- message
	return mock.MockOnData(message)
}

func (mock *MockAWSMessageListener) OnServiceIDs(message *aws.Message) error {
	mock.ChServiceIDsArg <- message
	return mock.MockServiceIDs(message)
}
