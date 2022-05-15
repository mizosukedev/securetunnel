package client

import (
	"context"
	"sync"
)

// workerManager is a structure for managing Worker for each StreamID.
type workerManager struct {
	workerMap map[int32]*Worker // map[StreamID]*Worker
	rwMutex   *sync.RWMutex     //
	bufSize   int               // channel buffer size
}

// start Worker.
func (mng *workerManager) start(streamID int32) *Worker {

	mng.rwMutex.Lock()
	defer mng.rwMutex.Unlock()

	worker := NewWorker(mng.bufSize)
	worker.Start(context.Background())
	mng.workerMap[streamID] = worker

	return worker
}

// exec Worker.
// Returns true if the corresponding StreamID exists, false otherwise.
func (mng *workerManager) exec(streamID int32, fnc func(context.Context)) bool {

	mng.rwMutex.RLock()
	defer mng.rwMutex.RUnlock()

	worker, ok := mng.workerMap[streamID]
	if ok {
		worker.Exec(fnc)
		return true
	}

	return false
}

// stop Worker asynchronously.
func (mng *workerManager) stop(streamID int32) {

	// The reason why Worker is stopped asynchronously is as follows.
	// - The Worker to stop may be Worker's goroutine. -> dead lock.
	// - The timing to stop the Worker does not have to be exact.
	go func() {

		mng.rwMutex.Lock()
		defer mng.rwMutex.Unlock()

		worker, ok := mng.workerMap[streamID]
		if ok {
			worker.Stop()
			delete(mng.workerMap, streamID)
		}
	}()
}

// getAll Worker.
func (mng *workerManager) getAll() []*Worker {

	mng.rwMutex.RLock()
	defer mng.rwMutex.RUnlock()

	workers := make([]*Worker, 0)

	for _, worker := range mng.workerMap {
		workers = append(workers, worker)
	}

	return workers
}

// stopAll Worker.
func (mng *workerManager) stopAll() {

	workers := mng.getAll()

	for _, worker := range workers {
		worker.Stop()
	}
}
