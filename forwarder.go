package haminer

// Forwarder define an interface to forward parsed HAProxy log to storage
// engine.
type Forwarder interface {
	Forwards(halogs []*HttpLog)
}
