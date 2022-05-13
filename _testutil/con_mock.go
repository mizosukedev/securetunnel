package testutil

import (
	"net"
	"time"
)

type MockConn struct {
	ChReadArgs     chan []byte
	MockRead       func(b []byte) (n int, err error)
	ChWriteArgs    chan []byte
	MockWrite      func(b []byte) (n int, err error)
	MockClose      func() error
	MockLocalAddr  func() net.Addr
	MockRemoteAddr func() net.Addr
}

func NewMockConn() *MockConn {

	mock := &MockConn{
		ChReadArgs: make(chan []byte, 10),
		MockRead: func(b []byte) (n int, err error) {
			return 0, nil
		},
		ChWriteArgs: make(chan []byte, 10),
		MockWrite: func(b []byte) (n int, err error) {
			return 0, nil
		},
		MockClose: func() error {
			return nil
		},
		MockLocalAddr: func() net.Addr {
			return nil
		},
		MockRemoteAddr: func() net.Addr {
			return nil
		},
	}

	return mock
}

func (mock *MockConn) Read(b []byte) (n int, err error) {
	mock.ChReadArgs <- b
	return mock.MockRead(b)
}

func (mock *MockConn) Write(b []byte) (n int, err error) {
	mock.ChWriteArgs <- b
	return mock.MockWrite(b)
}

func (mock *MockConn) Close() error {
	return mock.MockClose()
}

func (mock *MockConn) LocalAddr() net.Addr {
	return mock.MockLocalAddr()
}

func (mock *MockConn) RemoteAddr() net.Addr {
	return mock.MockRemoteAddr()
}

func (mock *MockConn) SetDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func (mock *MockConn) SetReadDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func (mock *MockConn) SetWriteDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}
