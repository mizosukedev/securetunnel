package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ListTunnelsRequest struct {
}

func (request ListTunnelsRequest) Validate() error {
	return nil
}

type ListTunnelsTunnel struct {
	ID            string       `json:"tunnelId"`
	Status        TunnelStatus `json:"status"`
	ThingName     string       `json:"thingName"`
	LastUpdatedAt time.Time    `json:"lastUpdatedAt"`
}

type ListTunnelsResponse struct {
	Tunnels []ListTunnelsTunnel `json:"tunnelSummaries"`
}

// ListTunnels returns list of tunnel information.
// Example:
// 	curl -s http://localhost:18080/tunnel/list
func (svc *Services) ListTunnels(ctx *gin.Context, args ListTunnelsRequest) {

	tunnels, err := svc.Store.GetTunnels(TunnelSearchOptions{})
	if err != nil {
		message := fmt.Sprintf("failed to get Tunnel: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	responseBody := ListTunnelsResponse{
		Tunnels: []ListTunnelsTunnel{},
	}

	for _, tunnel := range tunnels {
		responseTunnel := ListTunnelsTunnel{
			ID:            tunnel.ID,
			Status:        tunnel.Status(),
			ThingName:     tunnel.ThingName,
			LastUpdatedAt: tunnel.LastUpdatedAt,
		}
		responseBody.Tunnels = append(responseBody.Tunnels, responseTunnel)
	}

	ctx.JSON(http.StatusOK, responseBody)
}
