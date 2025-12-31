package fuego

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEWriter provides methods for streaming Server-Sent Events.
// Use Context.SSE() to obtain an SSEWriter.
//
// Example:
//
//	func Get(c *fuego.Context) error {
//	    sse, err := c.SSE()
//	    if err != nil {
//	        return err
//	    }
//
//	    for i := 0; i < 10; i++ {
//	        if sse.IsClosed() {
//	            break
//	        }
//	        sse.Send("message", fmt.Sprintf("Event %d", i))
//	        time.Sleep(time.Second)
//	    }
//	    return nil
//	}
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	closed  bool
}

// Send sends an SSE event with an optional event type.
// If event is empty, only the data field is sent.
// Returns an error if the connection is closed or write fails.
//
// Example:
//
//	sse.Send("message", "Hello, World!")
//	// Output: event: message\ndata: Hello, World!\n\n
func (s *SSEWriter) Send(event, data string) error {
	if s.closed {
		return fmt.Errorf("sse: connection closed")
	}

	var err error
	if event != "" {
		_, err = fmt.Fprintf(s.w, "event: %s\n", event)
		if err != nil {
			s.closed = true
			return err
		}
	}
	_, err = fmt.Fprintf(s.w, "data: %s\n\n", data)
	if err != nil {
		s.closed = true
		return err
	}
	s.flusher.Flush()
	return nil
}

// SendData sends data without an event type.
// Equivalent to Send("", data).
func (s *SSEWriter) SendData(data string) error {
	return s.Send("", data)
}

// SendJSON marshals the data to JSON and sends it as an SSE event.
// Returns an error if marshaling fails or the connection is closed.
//
// Example:
//
//	sse.SendJSON("user", User{Name: "Alice"})
//	// Output: event: user\ndata: {"name":"Alice"}\n\n
func (s *SSEWriter) SendJSON(event string, data any) error {
	if s.closed {
		return fmt.Errorf("sse: connection closed")
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sse: failed to marshal JSON: %w", err)
	}
	return s.Send(event, string(jsonBytes))
}

// SendComment sends an SSE comment.
// Comments are prefixed with ":" and are typically used for keep-alive pings.
//
// Example:
//
//	sse.SendComment("keep-alive")
//	// Output: : keep-alive\n\n
func (s *SSEWriter) SendComment(comment string) error {
	if s.closed {
		return fmt.Errorf("sse: connection closed")
	}

	_, err := fmt.Fprintf(s.w, ": %s\n\n", comment)
	if err != nil {
		s.closed = true
		return err
	}
	s.flusher.Flush()
	return nil
}

// SendRetry sets the reconnection time for the client in milliseconds.
// The client will wait this long before attempting to reconnect after a disconnect.
//
// Example:
//
//	sse.SendRetry(5000) // 5 seconds
//	// Output: retry: 5000\n\n
func (s *SSEWriter) SendRetry(milliseconds int) error {
	if s.closed {
		return fmt.Errorf("sse: connection closed")
	}

	_, err := fmt.Fprintf(s.w, "retry: %d\n\n", milliseconds)
	if err != nil {
		s.closed = true
		return err
	}
	s.flusher.Flush()
	return nil
}

// SendID sets the event ID.
// The client will send this ID in the Last-Event-ID header when reconnecting.
//
// Example:
//
//	sse.SendID("evt-123")
//	sse.Send("message", "data")
//	// Output: id: evt-123\nevent: message\ndata: data\n\n
func (s *SSEWriter) SendID(id string) error {
	if s.closed {
		return fmt.Errorf("sse: connection closed")
	}

	_, err := fmt.Fprintf(s.w, "id: %s\n", id)
	if err != nil {
		s.closed = true
		return err
	}
	return nil
}

// IsClosed returns true if the SSE connection has been closed.
// This can happen if the client disconnects or a write error occurs.
func (s *SSEWriter) IsClosed() bool {
	return s.closed
}

// Close marks the SSE writer as closed.
// Subsequent Send calls will return an error.
func (s *SSEWriter) Close() {
	s.closed = true
}
