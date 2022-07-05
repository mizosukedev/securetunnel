package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mizosukedev/securetunnel/aws"
	"github.com/mizosukedev/securetunnel/log"
)

type TunnelConnectArgs struct {
	Mode        aws.Mode    `form:"local-proxy-mode"`
	AccessToken AccessToken `header:"access-token"`
	ClientToken string      `header:"client-token"`
}

func (request TunnelConnectArgs) Validate() error {

	switch request.Mode {
	case aws.ModeDestination, aws.ModeSource:
	default:
		return fmt.Errorf("invalid mode '%s'", request.Mode)
	}

	return nil
}

// TunnelConnect is a WebAPI for localproxy to connect and establish a network tunnel.
func (svc *Services) TunnelConnect(ctx *gin.Context, args TunnelConnectArgs) {

	// TODO:refactor

	now := time.Now()

	tunnelID := args.AccessToken.TunnelID()

	tunnel, err := svc.Store.GetTunnel(tunnelID)
	if err != nil {
		message := fmt.Sprintf("invalid access-token: %v", err)
		ctx.Header(aws.HeaderKeyStatusReason, message)
		ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
		return
	}

	// check access-token
	token := tunnel.Token(args.Mode)
	otherMode := aws.OtherMode(args.Mode)
	otherToken := tunnel.Token(otherMode)

	if args.AccessToken != token {

		if args.AccessToken == otherToken {
			message := fmt.Sprintf(
				"access-token for %s mode was specified for the connection in %s mode",
				otherMode,
				args.Mode)
			ctx.Header(aws.HeaderKeyStatusReason, message)
			ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
			return
		} else {
			message := "invalid access-token"
			ctx.Header(aws.HeaderKeyStatusReason, message)
			ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
			return
		}
	}

	peer := tunnel.Peer(args.Mode)

	// check client-token
	if 0 < peer.NumOfConn {
		if peer.ClientToken != args.ClientToken {
			message := "invalid client-token"
			ctx.Header(aws.HeaderKeyStatusReason, message)
			ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
			return
		}
	}

	// Check if tunnel is opened
	if !tunnel.Open() {
		message := aws.StatusReasonTunnelClosed
		ctx.Header(aws.HeaderKeyStatusReason, message)
		ctx.JSON(http.StatusBadRequest, gin.H{"message": message})
		return
	}

	connectionID, err := svc.IDTokenGen.NewConnectionID(tunnelID)
	if err != nil {
		message := fmt.Sprintf("failed to generate ID: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	// Add connection
	connection := Connection{
		ID:            connectionID,
		Rev:           1,
		TunnelID:      tunnelID,
		Mode:          args.Mode,
		Status:        ConnStatusConnected,
		LastUpdatedAt: now,
	}

	err = svc.Store.AddConnection(connection)
	if err != nil {
		message := fmt.Sprintf("failed to update Connection: %v", err)
		ctx.Header(aws.HeaderKeyStatusReason, message)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	// Update Tunnel
	peer.ConnectionID = connectionID
	peer.ClientToken = args.ClientToken
	peer.NumOfConn = peer.NumOfConn + 1

	err = svc.Store.UpdateTunnel(tunnel)
	if err != nil {
		message := "failed to update Tunnel"
		ctx.Header(aws.HeaderKeyStatusReason, message)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	defer func() {
		// Update connection status, when disconnecting.
		connection, innerErr := svc.Store.GetConnection(connectionID)
		if innerErr != nil {
			log.Error(innerErr)
			return
		}

		connection.Status = ConnStatusDisconnected
		connection.LastUpdatedAt = time.Now()

		innerErr = svc.Store.UpdateConnection(connection)
		if innerErr != nil {
			log.Error(innerErr)
		}

		log.Infof("peer disconnected tunnelID=%s mode=%s", tunnelID, args.Mode)
	}()

	// --------------------------------------------
	//  Successful verification. Start websocket
	// --------------------------------------------

	upgrader := websocket.Upgrader{}
	// Only V2 protocol is supported.
	upgrader.Subprotocols = []string{aws.SubProtocolV2}

	wsCon, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		message := "failed to upgrade websocket"
		ctx.Header(aws.HeaderKeyStatusReason, message)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	wsCon.SetReadLimit(aws.MaxWebSocketFrameSize)

	defer wsCon.Close()

	// --------------------------------------------
	//  No header can be returned after this line.
	//  The error is returned with close code.
	// --------------------------------------------

	err = svc.peerConManager.connect(tunnelID, args.Mode, wsCon, tunnel.Services)
	if err != nil {
		log.Error(err)
		return
	}
}
