package goose

type Publisher interface {
	Publish(msg Message) error
}
