package log

import (
	"io"
	"sync"
)

// Broadcaster is an io.Writer that fans out every Write to all registered
// subscriber channels.  It is safe for concurrent use.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

// NewBroadcaster creates a ready-to-use Broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Write implements io.Writer.  Each write (typically one log line) is copied
// to every subscriber channel.  Slow subscribers are skipped (non-blocking
// send) so a stuck client never blocks the logger.
func (b *Broadcaster) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- buf:
		default:
			// subscriber too slow – drop the message
		}
	}
	return len(p), nil
}

// Subscribe registers a new subscriber and returns a buffered channel that
// will receive copies of every log line.  Call Unsubscribe when done.
func (b *Broadcaster) Subscribe() chan []byte {
	ch := make(chan []byte, 256)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
func (b *Broadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
	close(ch)
}

// compile-time check
var _ io.Writer = (*Broadcaster)(nil)
