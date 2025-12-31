package fuego

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSEWriter_Send(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	err := sse.Send("message", "hello")
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: message\n") {
		t.Errorf("Expected 'event: message', got: %s", body)
	}
	if !strings.Contains(body, "data: hello\n") {
		t.Errorf("Expected 'data: hello', got: %s", body)
	}
}

func TestSSEWriter_Send_NoEvent(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	err := sse.Send("", "hello")
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	body := w.Body.String()
	if strings.Contains(body, "event:") {
		t.Errorf("Send with empty event should not include event field, got: %s", body)
	}
	if !strings.Contains(body, "data: hello\n") {
		t.Errorf("Expected 'data: hello', got: %s", body)
	}
}

func TestSSEWriter_SendData(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	err := sse.SendData("hello")
	if err != nil {
		t.Fatalf("SendData() failed: %v", err)
	}

	body := w.Body.String()
	if strings.Contains(body, "event:") {
		t.Errorf("SendData should not include event field, got: %s", body)
	}
	if !strings.Contains(body, "data: hello\n") {
		t.Errorf("Expected 'data: hello', got: %s", body)
	}
}

func TestSSEWriter_SendJSON(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	data := map[string]string{"name": "Alice"}
	err := sse.SendJSON("user", data)
	if err != nil {
		t.Fatalf("SendJSON() failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: user\n") {
		t.Errorf("Expected 'event: user', got: %s", body)
	}
	if !strings.Contains(body, `"name":"Alice"`) {
		t.Errorf("Expected JSON data, got: %s", body)
	}
}

func TestSSEWriter_SendJSON_MarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	// Channels cannot be marshaled to JSON
	err := sse.SendJSON("test", make(chan int))
	if err == nil {
		t.Error("Expected error for non-marshalable value")
	}
	if !strings.Contains(err.Error(), "marshal") {
		t.Errorf("Expected marshal error, got: %v", err)
	}
}

func TestSSEWriter_SendComment(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	err := sse.SendComment("keep-alive")
	if err != nil {
		t.Fatalf("SendComment() failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, ": keep-alive\n") {
		t.Errorf("Expected ': keep-alive', got: %s", body)
	}
}

func TestSSEWriter_SendRetry(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	err := sse.SendRetry(5000)
	if err != nil {
		t.Fatalf("SendRetry() failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "retry: 5000\n") {
		t.Errorf("Expected 'retry: 5000', got: %s", body)
	}
}

func TestSSEWriter_SendID(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	err := sse.SendID("evt-123")
	if err != nil {
		t.Fatalf("SendID() failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "id: evt-123\n") {
		t.Errorf("Expected 'id: evt-123', got: %s", body)
	}
}

func TestSSEWriter_IsClosed(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	if sse.IsClosed() {
		t.Error("Expected IsClosed() to be false initially")
	}

	sse.Close()

	if !sse.IsClosed() {
		t.Error("Expected IsClosed() to be true after Close()")
	}
}

func TestSSEWriter_SendAfterClose(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	sse.Close()

	err := sse.Send("message", "hello")
	if err == nil {
		t.Error("Expected error when sending after close")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("Expected 'closed' in error message, got: %v", err)
	}
}

func TestSSEWriter_SendDataAfterClose(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	sse.Close()

	err := sse.SendData("hello")
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

func TestSSEWriter_SendJSONAfterClose(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	sse.Close()

	err := sse.SendJSON("test", map[string]string{"key": "value"})
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

func TestSSEWriter_SendCommentAfterClose(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	sse.Close()

	err := sse.SendComment("ping")
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

func TestSSEWriter_SendRetryAfterClose(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	sse.Close()

	err := sse.SendRetry(5000)
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

func TestSSEWriter_SendIDAfterClose(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	sse.Close()

	err := sse.SendID("123")
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

func TestContext_SSE(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	sse, err := c.SSE()
	if err != nil {
		t.Fatalf("SSE() failed: %v", err)
	}

	// Check headers
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got '%s'", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("Connection") != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got '%s'", w.Header().Get("Connection"))
	}
	if w.Header().Get("X-Accel-Buffering") != "no" {
		t.Errorf("Expected X-Accel-Buffering 'no', got '%s'", w.Header().Get("X-Accel-Buffering"))
	}

	// Check that context is marked as written
	if !c.Written() {
		t.Error("Expected context to be marked as written after SSE()")
	}

	// Send an event
	err = sse.Send("test", "data")
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: test\n") {
		t.Errorf("Expected 'event: test' in body, got: %s", body)
	}
}

func TestSSEWriter_MultipleEvents(t *testing.T) {
	w := httptest.NewRecorder()
	sse := &SSEWriter{w: w, flusher: w}

	_ = sse.Send("event1", "data1")
	_ = sse.Send("event2", "data2")
	_ = sse.SendData("data3")

	body := w.Body.String()

	if !strings.Contains(body, "event: event1\n") {
		t.Errorf("Expected 'event: event1', got: %s", body)
	}
	if !strings.Contains(body, "data: data1\n") {
		t.Errorf("Expected 'data: data1', got: %s", body)
	}
	if !strings.Contains(body, "event: event2\n") {
		t.Errorf("Expected 'event: event2', got: %s", body)
	}
	if !strings.Contains(body, "data: data2\n") {
		t.Errorf("Expected 'data: data2', got: %s", body)
	}
	if !strings.Contains(body, "data: data3\n") {
		t.Errorf("Expected 'data: data3', got: %s", body)
	}
}
