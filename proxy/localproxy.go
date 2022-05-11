package proxy

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/mizosukedev/securetunnel/client"
)

// LocalProxyOptions represents options of LocalProxy.
type LocalProxyOptions struct {

	// Mode represents local proxy mode(ModeSource or ModeDestination).
	Mode client.Mode

	// ServiceConfigs represents the information of the local services to connect to in destination mode,
	// and represents the information of the local services to listen in source mode.
	ServiceConfigs []ServiceConfig

	// Endpoint represents service endpoint.
	Endpoint *url.URL

	// Token represents token in the return value of AWS OpenTunnel WebAPI.
	Token string

	// TLSConfig If you want to customize tls.Config, set the value. Otherwise it sets null.
	TLSConfig *tls.Config

	// EndpointDialTimeout sets the timeout value when connecting to websocket.
	EndpointDialTimeout time.Duration

	// LocalDialTimeout sets the timeout value when connection to local service(ex. ssh server).
	LocalDialTimeout time.Duration

	// EndpointReconnectInterval represents the interval when reconnecting to endpoint.
	EndpointReconnectInterval time.Duration

	// PingInterval represents the interval when sending ping in websocket.
	PingInterval time.Duration
}

// LocalProxy represents localproxy in aws secure tunneling service.
type LocalProxy struct {
}

// Run the fllowing.
// 	[Source mode]
// 	- Start the local server listening using LocalProxyOptions.ServiceConfigs.
// 	- Connect to the tunneling service.
// 	- When the client(ssh etc.) connects, it sends StreamStart to the tunneling service.
// 	- Continue sending messages from client to the tunneling service and messages from the tunneling service to client.
// 	[Destination mode]
// 	- Connect to the tunneling service.
// 	- Connect to the local service using LocalProxyOptions.ServiceConfigs when StreamStart is received.
// 	- Continue sending messages from the local service to the tunneling service and messages from the tunneling service to the local service.
// This method is a blocking method.
// And this medhod does not return control to the caller until the following phenomenons occur.
// 	- Context is done.
// 	- http response status code is 400-499, when connecting to AWS.
// 	- Tunnel is closed.
func (proxy *LocalProxy) Run(ctx context.Context) error {
	return nil
}
