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
func (suite *WorkerTest) TestWorkerNormal() {

	bufSize := 10
	functionCount := 100

	worker := NewWorker(bufSize)
	worker.Start(context.Background())

	suite.Require().Equal(bufSize, cap(worker.exec))

	buf := make(chan int, functionCount)

	for i := 0; i < functionCount; i++ {
		data := i
		worker.Exec(func(ctx context.Context) {
			buf <- data
		})
	}

	for i := 0; i < functionCount; i++ {
		actual := <-buf
		expected := i

		suite.Require().Equal(expected, actual)
	}

	// Confirm no panic occurs even if Stop() is executed multiple times.
	worker.Stop()
	worker.Stop()
}

func (suite *WorkerTest) TestWorkerContextCancel() {

	worker := NewWorker(10)

	ctx, cancel := context.WithCancel(context.Background())
	go worker.Run(ctx)

	called := make(chan struct{})
	worker.Exec(func(ctx context.Context) {
		called <- struct{}{}
		<-ctx.Done()
	})

	<-called
	cancel()
	<-worker.terminate
	worker.Stop()
}
