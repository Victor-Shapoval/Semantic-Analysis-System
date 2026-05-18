package goose

type Message struct {
	GoCbRef string
	GoID    string // GOOSE identifier

	StNum uint32
	SqNum uint32

	Trip bool
}
