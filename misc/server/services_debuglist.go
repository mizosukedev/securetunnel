package server

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

type DebugListRequest struct {
}

type DebugListTunnel struct {
	Tunnel      Tunnel       `json:"tunnel"`
	Connections []Connection `json:"connections"`
}

func (request DebugListRequest) Validate() error {
	return nil
}

// DebugListTunnels returns list of tunnel information.
// Example:
// 	curl -s http://localhost:18080/debug/tunnel/list
func (svc *Services) DebugListTunnels(ctx *gin.Context, args DebugListRequest) {

	tunnels, err := svc.Store.GetTunnels(TunnelSearchOptions{})
	if err != nil {
		message := fmt.Sprintf("failed to get Tunnel: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	debugListTunnels := make([]DebugListTunnel, 0)

	for _, tunnel := range tunnels {

		connections, err := svc.Store.GetConnections(
			ConnectionSearchOptions{TunnelID: tunnel.ID})

		if err != nil {
			message := fmt.Sprintf("failed to get Connection: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
			return
		}

		debugListTunnel := DebugListTunnel{
			Tunnel:      tunnel,
			Connections: connections,
		}

		debugListTunnels = append(debugListTunnels, debugListTunnel)
	}

	sort.Slice(debugListTunnels, func(i, j int) bool {
		x := debugListTunnels[i].Tunnel
		y := debugListTunnels[j].Tunnel
		return x.CreatedAt.Before(y.CreatedAt)
	})

	ctx.JSON(http.StatusOK, debugListTunnels)
}
