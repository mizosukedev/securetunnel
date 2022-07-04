package server

import (
	"fmt"
	"net/http"

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
	IDTokenGen *IDTokenGen
	NeedAuth   bool
	Auth       Authorizer
	Store      Store
	Notifier   Notifier
}

func (svc *Services) Start() error {

	return nil
}
