package gyre

// EventType defines event type
type EventType int

// Event types
const (
	EventEnter EventType = iota + 1
	EventJoin
	EventLeave
	EventExit
	EventWhisper
	EventShout
)

// Converts EventType to string.
func (e EventType) String() string {
	switch e {
	case EventEnter:
		return "EventEnter"
	case EventJoin:
		return "EventJoin"
	case EventLeave:
		return "EventLeave"
	case EventExit:
		return "EventExit"
	case EventWhisper:
		return "EventWhisper"
	case EventShout:
		return "EventShout"
	}

	return ""
}

// Event represents an event which contains information about the sender and the
// group it belongs.
type Event struct {
	eventType EventType         // Event type
	sender    string            // Sender UUID as string
	name      string            // Sender public name as string
	address   string            // Sender ipaddress as string, for an ENTER event
	headers   map[string]string // Headers, for an ENTER event
	group     string            // Group name for a SHOUT event
	msg       []byte            // Message payload for SHOUT or WHISPER
}

// Type returns event type, which is a EventType.
func (e *Event) Type() EventType {
	return e.eventType
}

// Sender returns the sending peer's id as a string.
func (e *Event) Sender() string {
	return e.sender
}

// Name returns the sending peer's public name as a string.
func (e *Event) Name() string {
	return e.name
}

// Addr returns the sending peer's ipaddress as a string.
func (e *Event) Addr() string {
	return e.address
}

// Headers returns the event headers, or nil if there are none
func (e *Event) Headers() map[string]string {
	return e.headers
}

// Header returns value of a header from the message headers
// obtained by ENTER.
func (e *Event) Header(name string) (value string, ok bool) {
	value, ok = e.headers[name]
	return
}

// Group returns the group name that a SHOUT event was sent to.
func (e *Event) Group() string {
	return e.group
}

// Msg returns the incoming message payload (currently one frame).
func (e *Event) Msg() []byte {
	return e.msg
}
