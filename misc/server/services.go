package server

import "github.com/gin-gonic/gin"

type Services struct {
}

func (svc *Services) Init() error {

	return nil
}

func (svc *Services) Login(*gin.Context) {
}
func (svc *Services) Logout(*gin.Context) {
}
