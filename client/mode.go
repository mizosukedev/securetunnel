package client

// Mode represents local proxy mode.
type Mode string

const (
	// ModeSource represents source mode.
	ModeSource Mode = "source"
	// ModeDestination represents destination mode.
	ModeDestination Mode = "destination"
)
