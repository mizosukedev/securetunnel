package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mizosukedev/securetunnel/cmd/common"
	"github.com/mizosukedev/securetunnel/log"
	"github.com/mizosukedev/securetunnel/misc/server"
)

var (
	// TODO:command line arguments
	address  = "0.0.0.0:18080"
	logLevel = "debug"
)

func main() {

	err := common.SetupLogger(logLevel)
	if err != nil {
		fmt.Printf("failed to setup logger: %v", err)
		os.Exit(1)
	}

	// --------
	//  Route
	// --------
	engine := gin.Default()

	svc := &server.Services{}

	err = svc.Init()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	engine.POST("/login", svc.Login)
	engine.POST("/logout", svc.Logout)

	// -----
	//  Run
	// -----
	err = engine.Run(address)

	if err != nil {
		log.Errorf("http server stopped: %v", err)
	}
}
