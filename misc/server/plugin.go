package server

// Pluggable is an interface that indicates that the structure is pluggable.
type Pluggable interface {
	// Init is always executed only once before it is used.
	Init() error
}
