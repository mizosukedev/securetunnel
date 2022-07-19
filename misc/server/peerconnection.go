package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mizosukedev/securetunnel/aws"
	"github.com/mizosukedev/securetunnel/log"
)

// [#67] When making the server a scalable implementation, reimplement it here.

type peerConnectionKey struct {
	tunnelID string
	mode     aws.Mode
}

type connectionWriter interface {
	asyncWrite(message *aws.Message)
}

type nilConWriter struct {
}

func (*nilConWriter) asyncWrite(message *aws.Message) {
}

type peerConnection struct {
	rwMutex       *sync.RWMutex
	wsCon         *websocket.Conn
	serviceIDs    []string
	chMessage     chan *aws.Message
	associatedCon connectionWriter
	closed        bool
}

func newPeerConnection(con *websocket.Conn, serviceIDs []string) *peerConnection {

	instance := &peerConnection{
		rwMutex:       &sync.RWMutex{},
		wsCon:         con,
		serviceIDs:    serviceIDs,
		chMessage:     make(chan *aws.Message, 10),
		associatedCon: &nilConWriter{},
		closed:        false,
	}

	return instance
}

func (peerCon *peerConnection) run() error {

	err := peerCon.sendServiceIDs()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		err := peerCon.close(websocket.CloseNormalClosure, "normal closure")
		if err != nil {
			log.Error(err)
		}
		close(peerCon.chMessage)
	}()

	chReadErr := make(chan error)
	chWriteErr := make(chan error)

	go func() {
		chWriteErr <- peerCon.startWritingMessage(ctx)
	}()
	go func() {
		chReadErr <- peerCon.startReadingMessage(ctx)
	}()

	select {
	case err = <-chReadErr:
		cancel()
		<-chWriteErr
	case err = <-chWriteErr:
		cancel()
		closeErr := peerCon.close(websocket.CloseNormalClosure, "normal closure")
		if closeErr != nil {
			log.Error(closeErr)
		}
		<-chReadErr
	case <-ctx.Done():
	}

	return err
}

func (peerCon *peerConnection) sendServiceIDs() error {

	serviceIDsMsg := &aws.Message{
		Type:                aws.Message_SERVICE_IDS,
		AvailableServiceIds: peerCon.serviceIDs,
	}

	binServiceIDsMsg, err := aws.SerializeMessage(serviceIDsMsg)
	if err != nil {
		return err
	}

	err = peerCon.wsCon.WriteMessage(websocket.BinaryMessage, binServiceIDsMsg)
	if err != nil {
		err = fmt.Errorf("failed to send ServiceIDs message: %w", err)
		return err
	}

	return nil
}

func (peerCon *peerConnection) asyncWrite(message *aws.Message) {
	peerCon.chMessage <- message
}

func (peerCon *peerConnection) writeToAssociated(message *aws.Message) {
	peerCon.rwMutex.RLock()
	defer peerCon.rwMutex.RUnlock()

	peerCon.associatedCon.asyncWrite(message)
}

func (peerCon *peerConnection) startReadingMessage(ctx context.Context) error {

	binReader := aws.NewBReaderFromWSReader(peerCon.wsCon)
	reader := aws.NewMessageReader(binReader)

	for {

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		for {
			message, err := reader.Read()
			if err != nil {
				return err
			}

			// TODO: validate message? serviceID exists...etc

			peerCon.writeToAssociated(message)
		}

	}
}

func (peerCon *peerConnection) startWritingMessage(ctx context.Context) error {

	for {

		select {

		case <-ctx.Done():
			return nil

		case message := <-peerCon.chMessage:

			binMessage, err := aws.SerializeMessage(message)
			if err != nil {
				return err
			}

			err = peerCon.wsCon.WriteMessage(websocket.BinaryMessage, binMessage)
			if err != nil {
				return err
			}
		}
	}
}

func (peerCon *peerConnection) associate(conWriter connectionWriter) {
	peerCon.rwMutex.Lock()
	defer peerCon.rwMutex.Unlock()

	peerCon.associatedCon = conWriter
}

func (peerCon *peerConnection) close(closeCode int, text string) error {

	peerCon.rwMutex.Lock()
	defer peerCon.rwMutex.Unlock()

	if peerCon.closed {
		return nil
	}

	peerCon.closed = true

	message := websocket.FormatCloseMessage(closeCode, text)
	timeout := time.Now().Add(time.Second)
	err := peerCon.wsCon.WriteControl(websocket.CloseMessage, message, timeout)
	if err != nil {
		err = fmt.Errorf("failed to write close frame: %w", err)
		return err
	}

	err = peerCon.wsCon.Close()
	if err != nil {
		err = fmt.Errorf("failed to close connection: %w", err)
		return err
	}

	return err
}
