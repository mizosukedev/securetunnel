package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DescribeTunnelRequest struct {
	TunnelID string `form:"tunnelId" json:"tunnelId"`
}

func (request DescribeTunnelRequest) Validate() error {
	return nil
}

type DescribeTunnelConnectionState struct {
	Status        ConnStatus `json:"status"`
	LastUpdatedAt time.Time  `json:"lastUpdatedAt"`
}

type DescribeTunnelResponse struct {
	TunnelID       string                        `json:"tunnelId"`
	ThingName      string                        `json:"thingName"`
	TimeoutMinutes uint                          `json:"timeoutMinutes"`
	Payload        string                        `json:"payload"`
	Status         TunnelStatus                  `json:"status"`
	DstConnState   DescribeTunnelConnectionState `json:"destinationConnectionState"`
	SrcConnState   DescribeTunnelConnectionState `json:"sourceConnectionState"`
	LastUpdatedAt  time.Time                     `json:"lastUpdatedAt"`
	CreatedAt      time.Time                     `json:"createdAt"`
}

// DescribeTunnel returns detail tunnel information.
// Example:
// 	curl -s http://localhost:18080/tunnel/describe?tunnelId=<tunnelID>
func (svc *Services) DescribeTunnel(ctx *gin.Context, args DescribeTunnelRequest) {

	tunnel, err := svc.Store.GetTunnel(args.TunnelID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	responseBody := DescribeTunnelResponse{
		TunnelID:       tunnel.ID,
		ThingName:      tunnel.ThingName,
		TimeoutMinutes: tunnel.TimeoutMinutes,
		Payload:        tunnel.Payload,
		Status:         tunnel.Status(),
		DstConnState:   DescribeTunnelConnectionState{},
		SrcConnState:   DescribeTunnelConnectionState{},
		LastUpdatedAt:  tunnel.LastUpdatedAt,
		CreatedAt:      tunnel.CreatedAt,
	}

	dstConn, err := svc.Store.GetConnection(tunnel.DestinationPeer.ConnectionID)
	if err != nil {
		responseBody.DstConnState.Status = ConnStatusDisconnected
	} else {
		responseBody.DstConnState.Status = dstConn.Status
		responseBody.DstConnState.LastUpdatedAt = dstConn.LastUpdatedAt
	}

	srcConn, err := svc.Store.GetConnection(tunnel.SourcePeer.ConnectionID)
	if err != nil {
		responseBody.SrcConnState.Status = ConnStatusDisconnected
	} else {
		responseBody.SrcConnState.LastUpdatedAt = srcConn.LastUpdatedAt
		responseBody.SrcConnState.Status = srcConn.Status
	}

	ctx.JSON(http.StatusOK, responseBody)
}
