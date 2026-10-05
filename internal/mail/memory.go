package mail

import (
	"context"
	"sync"
)

// MemorySender keeps messages in memory instead of sending them. Tests use it
// to assert on what the application would have sent.
type MemorySender struct {
	mu       sync.Mutex
	messages []Message
}

func NewMemorySender() *MemorySender {
	return &MemorySender{}
}

func (s *MemorySender) Send(_ context.Context, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
	return nil
}

// Sent returns a copy of every message sent so far, oldest first.
func (s *MemorySender) Sent() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.messages...)
}
