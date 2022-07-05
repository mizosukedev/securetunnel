package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mizosukedev/securetunnel/aws"
)

type DebugSendRequest struct {
	TunnelID            string           `form:"tunnelId" json:"tunnelId"`
	Mode                aws.Mode         `form:"mode" json:"mode"`
	StreamID            int32            `form:"streamID" json:"streamID"`
	Type                aws.Message_Type `form:"type" json:"type"`
	Ignorable           bool             `form:"ignorable" json:"ignorable"`
	ServiceID           string           `form:"serviceID" json:"serviceID"`
	AvailableServiceIds []string         `form:"serviceIDs" json:"serviceIDs"`
	Payload             []byte           `form:"payload" json:"payload"`
}

func (request DebugSendRequest) Validate() error {
	return nil
}

// DebugSendToPeer send message to localproxy.
// You can interrupt and send a message to a specific peer.
// Example:
// 	curl -s -X POST -d 'tunnelId=<tunnelID>' -d 'mode=source' -d 'type=4' http://localhost:18080/debug/tunnel/send
func (svc *Services) DebugSendToPeer(ctx *gin.Context, args DebugSendRequest) {

	message := &aws.Message{
		StreamId:            args.StreamID,
		Type:                args.Type,
		Ignorable:           args.Ignorable,
		Payload:             args.Payload,
		ServiceId:           args.ServiceID,
		AvailableServiceIds: args.AvailableServiceIds,
	}

	err := svc.peerConManager.sendToPeer(args.TunnelID, args.Mode, message)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}
