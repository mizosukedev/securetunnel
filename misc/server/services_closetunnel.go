package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CloseTunnelRequest struct {
	Delete   bool   `form:"delete" json:"delete"`
	TunnelID string `form:"tunnelId" json:"tunnelId"`
}

func (request CloseTunnelRequest) Validate() error {
	return nil
}

type CloseTunnelResponse struct {
}

// CloseTunnel close tunnel and disconnect from both localproxy.
// Executing this method does not immediately disconnect from both localproxy.
// Example:
// 	curl -v -s -X PUT -d 'tunnelId=<tunnelID>' http://localhost:18080/tunnel/close
func (svc *Services) CloseTunnel(ctx *gin.Context, args CloseTunnelRequest) {

	tunnel, err := svc.Store.GetTunnel(args.TunnelID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	now := time.Now()

	tunnel.LifetimeAt = now
	tunnel.LastUpdatedAt = now

	err = svc.Store.UpdateTunnel(tunnel)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if args.Delete {
		err = svc.Store.DeleteTunnel(args.TunnelID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	response := &CloseTunnelResponse{}

	ctx.JSON(http.StatusOK, response)
}
