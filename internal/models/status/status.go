package status

type Status uint8

const (
	NEW = iota
	PROCESSING
	INVALID
	PROCESSED
)
