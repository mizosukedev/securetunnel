package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mizosukedev/securetunnel/log"
	"github.com/spf13/pflag"
)

const (
	// ------------------------
	//  Program description
	// ------------------------
	description = `Description:
  client application for aws secure tunneling service.

    source mode example:
      $ export MITRA_LOCALPROXY_TOKEN=<set source token>
      $ mitra_localproxy -e wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel -s 10022

    destination mode example:
      $ export MITRA_LOCALPROXY_TOKEN=<set destination token>
      $ mitra_localproxy -e wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel -d 22

`

	// ------------------------
	//  Parameter usage
	// ------------------------
	usageEndpoint = `environment variable:MITRA_LOCALPROXY_ENDPOINT
The endpoint of secure tunneling service including region.
Example:
  -e wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel
  -e wss://data.tunneling.iot.ap-northeast-1.amazonaws.com:443/tunnel`

	usageSourceServices = `environment variable:MITRA_LOCALPROXY_SOURCE_SERVICES
Start localproxy in source mode and listen for connections on the specified "[address:]port".
Specify only when starting in source mode.
When tunneling multiple services, specify the service "service name=[address:]port", separated by commas.
Example:
  -s 10022
  -s "SSH=10022, RDP=localhost:13389"`

	usageDestinationServices = `environment variable:MITRA_LOCALPROXY_DESTINATION_SERVICES
Start localproxy in destination mode and connect to the specified "[address:]port".
Specify only when starting in destination mode.
When tunneling multiple services, specify the service "name=address:port", separated by commas.
Example:
  -d 22
  -d "SSH=22, RDP=localhost:3389"`

	usageToken = `environment variable:MITRA_LOCALPROXY_TOKEN
Client access token. Specify the return value of AWS OpenTunnel WebAPI.
Specify it with an environment variable instead of specifying it with an argument.
Anyone can refer to startup argument.`

	usageSkipSSLVerify = `environment variable:MITRA_LOCALPROXY_SKIP_SSL_VERIFY
This program accepts any certificate presented by the server.`

	usageLogLevel = `log level. debug / info / warn / error`
)

type arguments struct {
	endpoint            string
	sourceServices      string
	destinationServices string
	token               string
	skipSSLVerify       bool
	logLevel            string
}

func (args *arguments) dump() {

	log.Debug("-------------- Arguments ---------------")

	pflag.VisitAll(func(f *pflag.Flag) {
		name := f.Name
		val := f.Value.String()

		if name == "token" && val != "" {
			val = "*****"
		}

		if val == "" {
			val = "[ empty ]"
		}

		log.Debugf("%-20s = %s", name, val)
	})

	log.Debug("----------------------------------------")
}

func setupCLI() arguments {

	args := arguments{}

	pflag.CommandLine.SortFlags = false

	pflag.StringVarP(
		&args.endpoint,
		"endpoint",
		"e",
		os.Getenv("MITRA_LOCALPROXY_ENDPOINT"),
		usageEndpoint)

	pflag.StringVarP(
		&args.sourceServices,
		"source-services",
		"s",
		os.Getenv("MITRA_LOCALPROXY_SOURCE_SERVICES"),
		usageSourceServices)

	pflag.StringVarP(
		&args.destinationServices,
		"destination-services",
		"d",
		os.Getenv("MITRA_LOCALPROXY_DESTINATION_SERVICES"),
		usageDestinationServices)

	pflag.StringVarP(
		&args.token,
		"token",
		"t",
		os.Getenv("MITRA_LOCALPROXY_TOKEN"),
		usageToken)

	skipVerify, err := strconv.ParseBool(os.Getenv("MITRA_LOCALPROXY_SKIP_SSL_VERIFY"))
	if err != nil {
		skipVerify = false
	}

	pflag.BoolVarP(
		&args.skipSSLVerify,
		"skip-ssl-verify",
		"k",
		skipVerify,
		usageSkipSSLVerify)

	pflag.StringVarP(
		&args.logLevel,
		"log",
		"",
		"info",
		usageLogLevel)

	pflag.Usage = func() {

		flagUsages := pflag.CommandLine.FlagUsages()

		strBuilder := &strings.Builder{}
		strBuilder.WriteString(description)
		strBuilder.WriteString("Flags:\n")
		strBuilder.WriteString(flagUsages)

		fmt.Fprint(os.Stderr, strBuilder.String())
	}

	// delete 'pflag: help requested'
	pflag.ErrHelp = errors.New("")

	pflag.Parse()

	return args
}
