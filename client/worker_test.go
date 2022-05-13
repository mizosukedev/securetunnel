package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type WorkerTest struct {
	suite.Suite
}

func TestWorker(t *testing.T) {
	suite.Run(t, new(WorkerTest))
}

// TestWorkerNormal confirm the operation when Worker is used normally.
func (suite *WorkerTest) TestNormal() {

	bufSize := 10
	functionCount := 100

	worker := NewWorker(bufSize)
	worker.Start(context.Background())

	suite.Require().Equal(bufSize, cap(worker.chExec))

	chBuf := make(chan int, functionCount)

	for i := 0; i < functionCount; i++ {
		data := i
		worker.Exec(func(ctx context.Context) {
			chBuf <- data
		})
	}

	for i := 0; i < functionCount; i++ {
		actual := <-chBuf
		expected := i

		suite.Require().Equal(expected, actual)
	}

	// Confirm no panic occurs even if Stop() is executed multiple times.
	worker.Stop()
	worker.Stop()
}

// TestContextCancel confirm that Worker stop if context is canceled.
func (suite *WorkerTest) TestContextCancel() {

	worker := NewWorker(10)

	ctx, cancel := context.WithCancel(context.Background())
	go worker.Run(ctx)

	chCalled := make(chan struct{})
	worker.Exec(func(ctx context.Context) {
		chCalled <- struct{}{}
		<-ctx.Done()
	})

	<-chCalled
	cancel()
	<-worker.chTerminate
	worker.Stop()
}
