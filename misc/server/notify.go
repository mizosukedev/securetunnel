package server

import "github.com/mizosukedev/securetunnel/log"

// Notifier is an interface to notify the device that the tunnel has opened and its token.
type Notifier interface {
	Pluggable
	// Notify the device of tunnel information.
	// This method is triggered when the tunnel is opened.
	Notify(Tunnel) error
}

// NilNotifier is an interface that implements Notifier.
// NilNotifier does not notify any device.
type NilNotifier struct {
}

func (notifier *NilNotifier) Init() error {
	return nil
}

func (notifier *NilNotifier) Notify(tunnel Tunnel) error {

	if tunnel.ThingName != "" {
		log.Infof("Does not notify the target device -> %s", tunnel.ThingName)
	}
	return nil
}
