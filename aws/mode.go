package aws

// Mode represents local proxy mode.
type Mode string

const (
	// ModeSource represents source mode.
	ModeSource Mode = "source"
	// ModeDestination represents destination mode.
	ModeDestination Mode = "destination"
	// ModeUnknown represents unknown mode.
	ModeUnknown Mode = ""
)
func OtherMode(mode Mode) Mode {

	switch mode {
	case ModeDestination:
		return ModeSource
	case ModeSource:
		return ModeDestination
	default:
		return ModeUnknown
	}
}
