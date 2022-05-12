package proxy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StreamIDGenTest struct {
	suite.Suite
}

func TestStreamIDGen(t *testing.T) {
	suite.Run(t, new(StreamIDGenTest))
}

// TestNormal confirm the operation when streamIDGen is used normally.
func (suite *StreamIDGenTest) TestNormal() {

	idGen := newStreamIDGen()

	idMap := map[int32]struct{}{}

	for i := 0; i < maxCounter; i++ {
		id := idGen.generate()
		idMap[id] = struct{}{}
	}

	suite.Require().Len(idMap, maxCounter)

	id := idGen.generate()

	// check counter reset.
	counter := id / 1000000
	suite.Require().Equal(int32(1), counter)
}
