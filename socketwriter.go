package echos

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Value any
	ErrCh chan error
}

type ThreadSafeSocketWriter struct {
	*websocket.Conn
	sync.Mutex
	msgCh   chan Message
	closeCh chan struct{}
	closed  bool
}

func NewThreadSafeSocketWriter(conn *websocket.Conn) *ThreadSafeSocketWriter {
	t := &ThreadSafeSocketWriter{
		Conn:    conn,
		msgCh:   make(chan Message, 100),
		closeCh: make(chan struct{}),
	}

	go t.writeLoop()

	return t
}

func (t *ThreadSafeSocketWriter) WriteJSON(v any) error {
	errCh := make(chan error, 1)

	select {
	case t.msgCh <- Message{Value: v, ErrCh: errCh}:
	case <-t.closeCh:
		return websocket.ErrCloseSent
	}

	select {
	case err := <-errCh:
		return err
	case <-t.closeCh:
		return websocket.ErrCloseSent
	}
}

func (t *ThreadSafeSocketWriter) writeLoop() {
	for {
		select {
		case msg := <-t.msgCh:
			t.Lock()
			err := t.Conn.WriteJSON(msg.Value)
			t.Unlock()

			if msg.ErrCh != nil {
				msg.ErrCh <- err
				close(msg.ErrCh)
			}

		case <-t.closeCh:
			return
		}
	}
}

func (t *ThreadSafeSocketWriter) Close() {
	t.Lock()
	defer t.Unlock()

	if !t.closed {
		t.closed = true
		close(t.closeCh)
	}
}

func (t *ThreadSafeSocketWriter) WriteWithContext(ctx context.Context, v any) error {
	errCh := make(chan error, 1)

	select {
	case t.msgCh <- Message{Value: v, ErrCh: errCh}:
	case <-t.closeCh:
		return websocket.ErrCloseSent
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-errCh:
		return err
	case <-t.closeCh:
		return websocket.ErrCloseSent
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *ThreadSafeSocketWriter) SetWriteDeadline(deadline time.Time) {
	t.Lock()
	defer t.Unlock()
	t.Conn.SetWriteDeadline(deadline)
}
