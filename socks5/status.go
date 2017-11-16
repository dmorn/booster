package socks5

import (
	"errors"
	"time"
)

// RegisterStatusListener subscribes a listener that will be notified
// when the proxy changes its state, for example from "IDLE" to "proxying".
// Returns an error if the id is already registered.
// It is safe to use from multiple goroutines.
func (s *Socks5) RegisterStatusListener(id string, c chan<- uint8) error {
	s.Lock()
	defer s.Unlock()

	if s.statusListeners == nil {
		s.statusListeners = make(map[string]chan<- uint8)
	}

	if _, ok := s.statusListeners[id]; ok {
		return errors.New("proxy status listener: <" + id + "> already registered")
	}

	s.statusListeners[id] = c
	return nil
}

// RemoveStatusListener removes the subscriber.
func (s *Socks5) RemoveStatusListener(id string) {
	s.Lock()
	defer s.Unlock()
	// first close the channel
	if c, ok := s.statusListeners[id]; ok {
		close(c)
	}

	delete(s.statusListeners, id)
	s.Unlock()
}

// setStatus updates proxy's status and notifies every listener of the change.
func (s *Socks5) setStatus(status uint8) {
	for key, c := range s.statusListeners {

		go func(k string, ch chan<- uint8) {
			// safely remove the listener if the channel was closed
			defer func() {
				if err := recover(); err != nil {
					// tryed to send to closed channel - remove listener
					s.Printf("proxy status listener: tryed to send on closed channel <" + k + ">")
					s.RemoveStatusListener(k)
				}
			}()

			select {
			case ch <- status: // could cause panic
				return
			case <-time.After(1 * time.Second):
				// deadline exceeded. Remove listener
				s.RemoveStatusListener(k)
			}
		}(key, c)

	}
}

// Port safely returns proxy's listening port.
func (s *Socks5) Port() int {
	s.Lock()
	defer s.Unlock()
	return s.port
}
