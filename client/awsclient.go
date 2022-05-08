package client

import (
	"context"
)

// AWSMessageListener is an interface representing event handlers to be fired
// when localproxy received message from secure tunneling service.
type AWSMessageListener interface {

	// OnStreamStart is an event handler that fires when a StreamStart message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#streamstart
	OnStreamStart(message *Message) error

	// OnStreamReset is an event handler that fires when a StreamReset message is received.
	// This method may be executed multiple times with the same stream ID.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#streamreset
	OnStreamReset(message *Message)

	// OnSessionReset is an event handler that fires when a SessionReset message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#sessionreset
	OnSessionReset(message *Message)

	// OnReceivedData is an event handler that fires when a Data message is received.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#data
	OnReceivedData(message *Message) error

	// OnReceivedServiceIDs is an event handler that fires when a ServiceIDs message is received.
	// 	Note:
	// 		The server will also send a ServiceIDs message when reconnecting.
	// 		That is, this method will also be executed when reconnecting.
	// 	See: https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md#serviceids
	OnReceivedServiceIDs(message *Message) error
}

// AWSClient is an interface, for the purpose of connectiong secure tunneling service.
type AWSClient interface {

	// Run keep websocket connection to secure tunneling service.
	// If the connection is successful, execute the following processing.
	// 	- Periodically send a ping frame to the service.
	// 	- Keep reading messages from the service and fire event handlers associated with AWSClientOptions.MessageListeners.
	// This method is a blocking method.
	// And this medhod does not return control to the caller until the following phenomenons occur.
	// 	- Caller context is done.
	// 	- http response status code is 400-499, when connecting to the service.
	// 	- Tunnel is closed.
	Run(ctx context.Context) error

	// SendStreamStart send StreamStart message to the service.
	// This method must be executed after Run() method.
	SendStreamStart(streamID int32, serviceID string) error

	// SendStreamReset send StreamReset message to the service.
	// This method must be executed after Run() method.
	SendStreamReset(streamID int32, serviceID string) error

	// SendData send Data message to the service.
	// This method must be executed after Run() method.
	SendData(streamID int32, serviceID string, data []byte) error
}

// AWSClientOptions is options of AWSClient.
type AWSClientOptions struct {
}

// NewAWSClient returns a instance which implements AWSClient.
func NewAWSClient(options AWSClientOptions) (AWSClient, error) {

	instance := &awsClient{}
	return instance, nil
}

// awsClient is a structure that implements AWSClient interface.
type awsClient struct {
}

// Run Refer to AWSClient.
func (client *awsClient) Run(ctx context.Context) error {

	return nil
}

// SendStreamStart Refer to AWSClient.
func (client *awsClient) SendStreamStart(streamID int32, serviceID string) error {

	return nil
}

// SendStreamReset Refer to AWSClient.
func (client *awsClient) SendStreamReset(streamID int32, serviceID string) error {

	return nil
}

// SendData Refer to AWSClient.
func (client *awsClient) SendData(streamID int32, serviceID string, data []byte) error {

	return nil
}
