package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/mizosukedev/securetunnel/cmd/common"
	"github.com/mizosukedev/securetunnel/log"
	"github.com/mizosukedev/securetunnel/misc/server"
)

var (
	// TODO:command line arguments
	needAuth  = false
	address   = "0.0.0.0:18080"
	logLevel  = "debug"
	debugMode = true
)

func main() {

	err := common.SetupLogger(logLevel)
	if err != nil {
		fmt.Printf("failed to setup logger: %v", err)
		os.Exit(1)
	}

	// -------------------------------
	//  TODO:open/init each libraries
	// -------------------------------

	var authorizer server.Authorizer

	authorizer = &server.NilAuthorizer{}

	pluggables := []server.Pluggable{authorizer}
	for _, pluggable := range pluggables {
		err := pluggable.Init()
		if err != nil {
			t := reflect.TypeOf(pluggable)
			log.Errorf("failed to init library %v: %v", t, err)
			os.Exit(1)
		}
	}

	// --------
	//  Route
	// --------
	engine := gin.Default()

	svc := &server.Services{
		NeedAuth: needAuth,
		Auth:     authorizer,
	}

	err = svc.Start()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	engine.POST("/login", svc.Login)

	needAuthGroup := engine.Group("/")
	needAuthGroup.Use(svc.AuthFilter)
	{
		needAuthGroup.POST("/logout", svc.Logout)

		tunnelGroup := needAuthGroup.Group("/tunnel")
		{
			tunnelGroup.POST("/open")
			tunnelGroup.GET("/list")
			tunnelGroup.GET("/describe")
			tunnelGroup.PUT("/close")
		}

		if debugMode {
			debugGroup := needAuthGroup.Group("/debug")
			{
				debugGroup.GET("/tunnel/list")
				debugGroup.POST("/tunnel/send")
			}
		}
	}

	engine.GET("/tunnel/connect")

	// -----
	//  Run
	// -----
	err = engine.Run(address)

	if err != nil {
		log.Errorf("http server stopped: %v", err)
	}
}
