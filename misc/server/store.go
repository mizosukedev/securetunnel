package server

type TunnelSearchOptions struct {
	// TODO: append
}

type ConnectionSearchOptions struct {
	TunnelID string
	// TODO: append
}

// Store is an interface for storing and retrieving data.
type Store interface {
	Pluggable
	GetTunnel(tunndlID string) (Tunnel, error)
	GetTunnels(options TunnelSearchOptions) ([]Tunnel, error)
	AddTunnel(tunnel Tunnel) error
	UpdateTunnel(tunnel Tunnel) error
	DeleteTunnel(tunndlID string) error
	GetConnection(connectionID ConnectionID) (Connection, error)
	GetConnections(options ConnectionSearchOptions) ([]Connection, error)
	AddConnection(connection Connection) error
	UpdateConnection(connection Connection) error
	DeleteConnection(connectionID ConnectionID) error
	//Truncate() error
}
