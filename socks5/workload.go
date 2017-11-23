package socks5

import (
	"errors"
	"time"
)

// RegisterWorkloadListener subscribes a listener that will be notified
// when the proxy changes its workload value.
// Returns an error if the id is already registered.
// It is safe to use from multiple goroutines.
func (s *Socks5) RegisterWorkloadListener(id string, c chan int) error {
	s.Lock()
	defer s.Unlock()

	if s.workloadListeners == nil {
		s.workloadListeners = make(map[string]chan int)
	}

	if _, ok := s.workloadListeners[id]; ok {
		return errors.New("proxy status listener: <" + id + "> already registered")
	}

	s.workloadListeners[id] = c
	return nil
}

// RemoveWorkloadListener removes the subscriber.
func (s *Socks5) RemoveWorkloadListener(id string) {
	s.Lock()
	defer s.Unlock()
	// first close the channel
	if c, ok := s.workloadListeners[id]; ok {
		close(c)
	}

	delete(s.workloadListeners, id)
}

// Port safely returns proxy's listening port.
func (s *Socks5) Port() int {
	s.Lock()
	defer s.Unlock()
	return s.port
}

func (s *Socks5) pushLoad() {
	s.Lock()
	s.workload++
	s.Unlock()

	s.workloadChanged()
}

func (s *Socks5) popLoad() {
	s.Lock()
	s.workload--
	// should never become negative
	if s.workload < 0 {
		s.workload = 0
	}
	s.Unlock()

	s.workloadChanged()
}

func (s *Socks5) workloadChanged() {
	for key, c := range s.workloadListeners {
		go func(k string, ch chan int) {
			// safely remove the listener if the channel was closed
			defer func() {
				if err := recover(); err != nil {
					// tryed to send to closed channel - remove listener
					s.Printf("proxy status listener: tryed to send on closed channel <" + k + ">")
					s.RemoveWorkloadListener(k)
				}
			}()

			s.Lock()
			wl := s.workload
			s.Unlock()

			s.Printf("workload: %v", wl)

			select {
			case ch <- wl: // could cause panic
				return
			case <-time.After(1 * time.Second):
				// deadline exceeded. Remove listener
				s.RemoveWorkloadListener(k)
			}
		}(key, c)

	}
}
