package proxy

import (
	"strconv"
	"sync"
	"time"
)

const (
	maxCounter = 2147
)

// streamIDGen is a structure that generate new StreamID.
// To ensure the uniqueness of StreamID, issue StreamID by combining time and counter.
// math.MaxInt32 = 2147483647
// 2147(counter field) | 483647(time field)
// Since time is only hours, minutes, and seconds, it falls within the range of 483647.
// 235959(max time) < 483647
// There is no problem if the number of connections is 2147 or less per second,
// but if it is more than that, duplicate StreamIDs will occur.
// I want AWS to implement StreamID to be issued from the server...
type streamIDGen struct {
	counterMutex *sync.Mutex
	counter      int
}

func newStreamIDGen() *streamIDGen {

	instance := &streamIDGen{
		counterMutex: &sync.Mutex{},
		counter:      0,
	}

	return instance
}

func (idGen *streamIDGen) generate() int32 {

	idGen.counterMutex.Lock()
	defer idGen.counterMutex.Unlock()

	if idGen.counter == maxCounter {
		idGen.counter = 0
	}

	idGen.counter++

	strTime := time.Now().Format("150405")
	time, _ := strconv.Atoi(strTime)

	counter := idGen.counter * 1000000
	id := counter + time

	return int32(id)
}
