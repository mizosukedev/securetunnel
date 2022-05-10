package client

import "context"

// Worker is a structure that has one channel and manages one goroutine.
// You can write function to the channel using Exec().
// The function is executed sequentially by the internal goroutine.
// This structure cannot be reused.
type Worker struct {
	exec      chan func(context.Context)
	terminate chan struct{}
	cancel    chan context.CancelFunc
}

// NewWorker returns a Worker instance.
// bufSize -> channel buffer size.
func NewWorker(bufSize int) *Worker {

	instance := &Worker{
		exec:      make(chan func(context.Context), bufSize),
		terminate: make(chan struct{}, 1),
		cancel:    make(chan context.CancelFunc, 1),
	}

	return instance
}

// Start goroutine and execute functions in channel sequentially.
func (worker *Worker) Start(ctx context.Context) {

	ctx, cancel := context.WithCancel(ctx)
	worker.cancel <- cancel

	go func() {
		defer close(worker.terminate)

		for {
			select {
			case <-ctx.Done():
				return
			case fnc := <-worker.exec:
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
	<-worker.terminate
}

// Exec is a function for writing functions to be executed sequentially.
func (worker *Worker) Exec(fnc func(context.Context)) {
	worker.exec <- fnc
}

// Stop inner goroutine.
func (worker *Worker) Stop() {

	cancel, ok := <-worker.cancel

	if ok {
		close(worker.cancel)
		cancel()
		<-worker.terminate
	}
}
