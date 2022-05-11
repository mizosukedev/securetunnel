package testutil

import "net"

type MockListener struct {
	MockAccept func() (net.Conn, error)
	MockClose  func() error
	MockAddr   func() net.Addr
}

func NewMockListener() *MockListener {

	mock := &MockListener{
		MockAccept: func() (net.Conn, error) {
			con := NewMockConn()
			return con, nil
		},
		MockClose: func() error {
			return nil
		},
		MockAddr: func() net.Addr {
			return &net.TCPAddr{}
		},
	}

	return mock
}

func (mock *MockListener) Accept() (net.Conn, error) {
	return mock.MockAccept()
}

func (mock *MockListener) Close() error {
	return mock.MockClose()
}

func (mock *MockListener) Addr() net.Addr {
	return mock.MockAddr()
}
