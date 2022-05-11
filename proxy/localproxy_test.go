package proxy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LocalProxyTest struct {
	suite.Suite
}

func ExampleLocalProxy() {

	options := LocalProxyOptions{
		// init fields
	}

	localProxy, err := NewLocalProxy(options)
	if err != nil {
		fmt.Println(err)
		return
	}

	// call cancel() in signal handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localProxy.Run(ctx)
}

func TestLocalProxy(t *testing.T) {
	suite.Run(t, new(LocalProxyTest))
}
