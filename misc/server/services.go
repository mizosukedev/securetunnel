package server

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type Validatable interface {
	Validate() error
}

func PreProcess[T any](handler func(*gin.Context, T)) gin.HandlerFunc {

	wrapHandler := func(ctx *gin.Context) {

		// new(T) -> **OpenTunnelRequest
		args := new(T)

		err := ctx.Bind(args)
		if err != nil {
			message := fmt.Sprintf("failed to parse request: %v", err)
			ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
			return
		}

		err = ctx.BindHeader(args)
		if err != nil {
			message := fmt.Sprintf("failed to parse header: %v", err)
			ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
			return
		}

		validatable, ok := any(*args).(Validatable)
		if ok {
			err = validatable.Validate()
			if err != nil {
				message := fmt.Sprintf("failed to validate request: %v", err)
				ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
				return
			}
		}

		handler(ctx, *args)
	}

	return wrapHandler
}

type Services struct {
	IDTokenGen         *IDTokenGen
	NeedAuth           bool
	InterServerNetwork string
	InterServerAddress string
	Auth               Authorizer
	Store              Store
	Notifier           Notifier
	peerConManager     *peerConManager
}

func (svc *Services) Start() error {

	// TODO: start GRPC inter server

	svc.peerConManager = &peerConManager{
		rwMutex:    &sync.RWMutex{},
		store:      svc.Store,
		peerConMap: map[peerConnectionKey]*peerConnection{},
	}

	return nil
}
