package gyre

type EventType int

const (
	EventEnter EventType = iota + 1
	EventJoin
	EventLeave
	EventExit
	EventWhisper
	EventShout
)

// Converts EventType to string
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

type Event struct {
	eventType EventType         // Event type
	sender    string            // Sender UUID as string
	name      string            // Sender public name as string
	address   string            // Sender ipaddress as string, for an ENTER event
	headers   map[string]string // Headers, for an ENTER event
	group     string            // Group name for a SHOUT event
	msg       []byte            // Message payload for SHOUT or WHISPER
}

// Returns event type, which is a EventType.
func (e *Event) Type() EventType {
	return e.eventType
}

// Returns the sending peer's id as a string.
func (e *Event) Sender() string {
	return e.sender
}

// Returns the sending peer's public name as a string.
func (e *Event) Name() string {
	return e.name
}

// Returns the sending peer's ipaddress as a string.
func (e *Event) Addr() string {
	return e.address
}

// Returns the event headers, or nil if there are none
func (e *Event) Headers() map[string]string {
	return e.headers
}

// Returns value of a header from the message headers
// obtained by ENTER.
func (e *Event) Header(name string) (value string, ok bool) {
	value, ok = e.headers[name]
	return
}

// Returns the group name that a SHOUT event was sent to.
func (e *Event) Group() string {
	return e.group
}

// Returns the incoming message payload (currently one frame).
func (e *Event) Msg() []byte {
	return e.msg
}
