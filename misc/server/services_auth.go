package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (svc *Services) Login(ctx *gin.Context) {

	svc.Auth.Login(ctx.Writer, ctx.Request)
}

func (svc *Services) Logout(ctx *gin.Context) {

	svc.Auth.Logout(ctx.Writer, ctx.Request)
}

func (svc *Services) AuthFilter(ctx *gin.Context) {

	if !svc.NeedAuth {
		return
	}

	err := svc.Auth.HasAuth(ctx.Request)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		ctx.Abort()
	}
}
