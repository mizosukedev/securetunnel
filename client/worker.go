package client

import "context"

// Worker is a structure that has one channel and manages one goroutine.
// You can write function to the channel using Exec().
// The function is executed sequentially by the internal goroutine.
// This structure cannot be reused.
type Worker struct {
	chExec      chan func(context.Context)
	chTerminate chan struct{}
	chCancel    chan context.CancelFunc
}

// NewWorker returns a Worker instance.
// bufSize -> channel buffer size.
func NewWorker(bufSize int) *Worker {

	instance := &Worker{
		chExec:      make(chan func(context.Context), bufSize),
		chTerminate: make(chan struct{}, 1),
		chCancel:    make(chan context.CancelFunc, 1),
	}

	return instance
}

// Start goroutine and execute functions in channel sequentially.
func (worker *Worker) Start(ctx context.Context) {

	ctx, cancel := context.WithCancel(ctx)
	worker.chCancel <- cancel

	go func() {
		defer close(worker.chTerminate)

		for {
			select {
			case <-ctx.Done():
				return
			case fnc := <-worker.chExec:
				fnc(ctx)
			}
		}
	}()
}

// Run is a method that behaves like Start().
// This medhod does not return control to the caller until the following phenomenons occur.
// 	- caller context is done.
// 	- Stop() method.
func (worker *Worker) Run(ctx context.Context) {
	worker.Start(ctx)
	<-worker.chTerminate
}

// Exec is a function for writing functions to be executed sequentially.
func (worker *Worker) Exec(fnc func(context.Context)) {
	worker.chExec <- fnc
}

// Stop inner goroutine.
func (worker *Worker) Stop() {

	cancel, ok := <-worker.chCancel

	if ok {
		close(worker.chCancel)
		cancel()
		<-worker.chTerminate
	}
}
