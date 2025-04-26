package channels

type OneWayChannel interface {
	Send(eventType string, data interface{}) error
	SendEndpoint(endpoint string) error
	Close()
}
