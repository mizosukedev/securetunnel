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
	needAuth          = false
	address           = "0.0.0.0:18080"
	tunnelIDDigit     = uint(3)
	tokenDigit        = uint(3)
	connectionIDDigit = uint(3)
	logLevel          = "debug"
	debugMode         = true
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
	var store server.Store
	var notifier server.Notifier

	authorizer = &server.NilAuthorizer{}
	store = &server.MemoryStore{}
	notifier = &server.NilNotifier{}

	pluggables := []server.Pluggable{authorizer, store, notifier}
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
		IDTokenGen: &server.IDTokenGen{
			TunnelIDDigit:     tunnelIDDigit,
			TokenDigit:        tokenDigit,
			ConnectionIDDigit: connectionIDDigit,
		},
		NeedAuth: needAuth,
		Auth:     authorizer,
		Store:    store,
		Notifier: notifier,
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
			tunnelGroup.POST("/open", server.PreProcess(svc.OpenTunnel))
			tunnelGroup.GET("/list", server.PreProcess(svc.ListTunnels))
			tunnelGroup.GET("/describe", server.PreProcess(svc.DescribeTunnel))
			tunnelGroup.PUT("/close", server.PreProcess(svc.CloseTunnel))
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
