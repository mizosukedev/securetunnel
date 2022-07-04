package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type OpenTunnelRequest struct {
	Services       []string `form:"services" json:"services"`
	ThingName      string   `form:"thingName" json:"thingName"`
	TimeoutMinutes uint     `form:"timeoutMinutes" json:"timeoutMinutes"`
	Payload        string   `form:"payload" json:"payload"`
}

func (request OpenTunnelRequest) Validate() error {
	return nil
}

type OpenTunnelResponse struct {
	TunnelID         string      `json:"tunnelId"`
	DestinationToken AccessToken `json:"destinationAccessToken"`
	SourceToken      AccessToken `json:"sourceAccessToken"`
}

// OpenTunnel creates a new tunnel and returns client access tokens.
// These tokens are used by the localproxy as credentials when connecting.
// Example:
// 	curl -s -X POST -H "Content-Type: application/json" -d '{"services": ["SSH", "RDP"], "thingName": "device1"}' http://localhost:18080/tunnel/open
// 	curl -s -X POST -d 'thingName=device' -d 'services=SSH' -d 'services=RDP' http://localhost:18080/tunnel/open
func (svc *Services) OpenTunnel(ctx *gin.Context, args OpenTunnelRequest) {

	ids, err := svc.IDTokenGen.NewIDTokens()
	if err != nil {
		message := fmt.Sprintf("failed to generate ID: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	if args.TimeoutMinutes == 0 {
		args.TimeoutMinutes = 720
	}

	now := time.Now()

	lifetime := now.Add(time.Duration(args.TimeoutMinutes) * time.Minute)

	tunnel := Tunnel{
		ID:               ids.TunnelID,
		Rev:              1,
		SourceToken:      ids.SourceToken,      // to hash?
		DestinationToken: ids.DestinationToken, // to hash?
		Services:         args.Services,
		ThingName:        args.ThingName,
		TimeoutMinutes:   args.TimeoutMinutes,
		SourcePeer:       &Peer{},
		DestinationPeer:  &Peer{},
		Payload:          args.Payload,
		LifetimeAt:       lifetime,
		CreatedAt:        now,
		LastUpdatedAt:    now,
	}

	err = svc.Store.AddTunnel(tunnel)
	if err != nil {
		message := fmt.Sprintf("failed to store Tunnel information: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": message})
		return
	}

	// notification
	err = svc.Notifier.Notify(tunnel)
	if err != nil {
		err = fmt.Errorf("failed to notify a device of Tunnel information: %w", err)

		deleteErr := svc.Store.DeleteTunnel(tunnel.ID)
		if deleteErr != nil {
			err = fmt.Errorf("failed to delete Tunnel information: %w", err)
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	responseBody := &OpenTunnelResponse{
		TunnelID:         tunnel.ID,
		DestinationToken: tunnel.DestinationToken,
		SourceToken:      tunnel.SourceToken,
	}

	// success
	ctx.JSON(http.StatusOK, responseBody)
}
