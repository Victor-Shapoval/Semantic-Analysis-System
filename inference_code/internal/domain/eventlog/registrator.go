package eventlog

type Registrator interface {
	Register(event FaultEvent) error
}
