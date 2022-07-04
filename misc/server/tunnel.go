package server

import (
	"time"

	"github.com/mizosukedev/securetunnel/aws"
)

type TunnelStatus string
type ConnStatus string

const (
	TunnelStatusUnknown = ""
	TunnelStatusOpen    = "OPEN"
	TunnelStatusClosed  = "CLOSED"

	ConnStatusUnknown      = ""
	ConnStatusConnected    = "CONNECTED"
	ConnStatusDisconnected = "DISCONNECTED"
)

type Tunnel struct {
	ID               string
	Rev              uint
	SourceToken      AccessToken
	SourcePeer       *Peer
	DestinationToken AccessToken
	DestinationPeer  *Peer
	Services         []string
	ThingName        string
	TimeoutMinutes   uint
	Payload          string
	LifetimeAt       time.Time
	CreatedAt        time.Time
	LastUpdatedAt    time.Time
}

func (tunnel Tunnel) Open() bool {

	result := tunnel.Status() == TunnelStatusOpen
	return result
}

func (tunnel Tunnel) Status() TunnelStatus {

	now := time.Now()
	// now < lifetime
	expired := tunnel.LifetimeAt.Before(now)

	if expired {
		return TunnelStatusClosed
	} else {
		return TunnelStatusOpen
	}
}

type Peer struct {
	ConnectionID ConnectionID
	ClientToken  string
	NumOfConn    int
}

type Connection struct {
	ID                 ConnectionID
	Rev                uint
	TunnelID           string
	Mode               aws.Mode
	Status             ConnStatus
	InterServerNetwork string // for scaling
	InterServerAddress string // for scaling
	LastUpdatedAt      time.Time
}
