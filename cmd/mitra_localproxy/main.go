package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/mizosukedev/securetunnel/client"
	"github.com/mizosukedev/securetunnel/log"
	"github.com/mizosukedev/securetunnel/proxy"
	"github.com/spf13/pflag"
)

const (
	endpointDialTimeout       = time.Second * 10
	localDialTimeout          = time.Second * 5
	endpointReconnectInterval = time.Second * 5
	pingInterval              = time.Second * 30
	tunnelMaxLifeTime         = time.Hour * 12
	localProxyMaxLifeTime     = tunnelMaxLifeTime + time.Hour
	terminateWaitTime         = time.Second * 30
)

func main() {

	args := setupCLI()

	err := setupLogger(args.logLevel)
	exitOnError(err)

	args.dump()

	options, err := createProxyOptions(args)
	exitOnError(err)

	err = options.Validate()
	exitOnError(err)

	if os.Getenv("MITRA_LOCALPROXY_TOKEN") == "" {
		log.Warn("token was specified in startup argument")
		log.Warn("Specify the token using environment variable instead of startup argument")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// signal handler
	applicationExit(func(signal os.Signal) {
		log.Infof(
			"received signal. Please wait up to %d seconds to terminate.",
			int(terminateWaitTime.Seconds()))

		cancel()

		go func() {
			<-time.After(terminateWaitTime)
			log.Warnf(
				"mitra_localproxy did not terminate within %d seconds, so it will be terminated.",
				int(terminateWaitTime.Seconds()))
			os.Exit(1)
		}()
	})

	// Run LocalProxy
	chTerminate := make(chan struct{})
	go func() {
		defer close(chTerminate)

		localProxy, err := proxy.NewLocalProxy(options)
		if err != nil {
			log.Error(err)
			return
		}

		log.Infof("start mitra_localproxy in %v mode", options.Mode)
		err = localProxy.Run(ctx)
		if err != nil {
			log.Error(err)
		}
	}()

	// Wait for LocalProxy.Run() to terimnate.
	select {
	// terminated gracefully.
	case <-chTerminate:
		log.Info("mitra_localproxy stopped")
	// Since LocalProxy did not terminate normally, so kill it.
	case <-time.After(localProxyMaxLifeTime):
		log.Error("Since mitra_localproxy did not terminate normally, so kill it")
		os.Exit(1)
	}
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		pflag.Usage()
		os.Exit(1)
	}
}

func createProxyOptions(args arguments) (proxy.LocalProxyOptions, error) {

	options := proxy.LocalProxyOptions{
		EndpointDialTimeout:       endpointDialTimeout,
		LocalDialTimeout:          localDialTimeout,
		EndpointReconnectInterval: endpointReconnectInterval,
		PingInterval:              pingInterval,
	}

	if args.endpoint == "" {
		err := errors.New("endpoint is required")
		return options, err
	}

	url, err := url.Parse(args.endpoint)
	if err != nil {
		return options, fmt.Errorf("endpoint is invalid: %v", err)
	}

	options.Endpoint = url

	// -------------------------

	if args.token == "" {
		err := errors.New("env var $MITRA_LOCALPROXY_TOKEN is required")
		return options, err
	}

	options.Token = args.token

	// -------------------------

	if args.sourceServices == "" && args.destinationServices == "" {
		err := errors.New("Specify only one of --source-services or --destination-services")
		return options, err
	}

	if args.sourceServices != "" && args.destinationServices != "" {
		err := errors.New("Specify only one of --source-services or --destination-services")
		return options, err
	}

	if args.sourceServices != "" {
		options.Mode = client.ModeSource
		options.ServiceConfigs, err = proxy.ParseServiceConfig(args.sourceServices)
		if err != nil {
			return options, err
		}
	}

	if args.destinationServices != "" {
		options.Mode = client.ModeDestination
		options.ServiceConfigs, err = proxy.ParseServiceConfig(args.destinationServices)
		if err != nil {
			return options, err
		}
	}

	// -------------------------

	tlsConfig := &tls.Config{
		InsecureSkipVerify: args.skipSSLVerify,
	}

	options.TLSConfig = tlsConfig

	return options, nil
}
