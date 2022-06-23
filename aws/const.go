package aws

import "errors"

const (
	QueryKeyProxyMode        = "local-proxy-mode"
	SubProtocolV2            = "aws.iot.securetunneling-2.0"
	HeaderKeyAccessToken     = "access-token"
	HeaderKeyClientToken     = "client-token"
	HeaderKeyStatusReason    = "X-Status-Reason"
	StatusReasonTunnelClosed = "Tunnel is closed"
	SizeOfMessageSize        = 2
	MaxWebSocketFrameSize    = 131076
)

var (
	//lint:ignore ST1005 user message use alone.
	ErrTunnelClosed = errors.New("Tunnel is closed")

	SubProtocols = []string{
		SubProtocolV2,
	}
)
