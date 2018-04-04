package pubsub

import (
	"fmt"
	"sync"
	"time"
)

type channel struct {
	sendc chan interface{}
	stopc chan interface{}

	sync.Mutex
	active bool
	ch     chan interface{}
}

func (c *channel) run() (chan interface{}, error) {
	if c.isActive() {
		return nil, fmt.Errorf("channel: unable to run: already active")
	}

	// create output channel
	c.Lock()
	c.ch = make(chan interface{})
	c.active = true
	c.Unlock()

	send := func(m interface{}) {
		defer func() {
			if err := recover(); err != nil {
				// we tried to send somethign trough ch and we found it closed.
				// need to remove this channel.
				c.stop()
			}
		}()

		select {
		case c.out() <- m:
		case <-time.After(time.Second):
			c.stop()
		}
	}

	go func() {
		for {
			select {
			case m := <-c.sendc:
				go send(m)
			case <-c.stopc:
				return
			}
		}
	}()

	return c.out(), nil
}

func newChannel() *channel {
	return &channel{
		sendc:  make(chan interface{}),
		stopc:  make(chan interface{}),
		active: false,
	}
}

func (c *channel) send(m interface{}) {
	if c.isActive() {
		c.sendc <- m
	}
}

func (c *channel) stop() {
	if !c.isActive() {
		return
	}

	c.stopc <- struct{}{} // stop run()
	c.setIsActive(false)  // set inactive so we don't forward messages anymore
	c.Lock()
	closeChanSafe(c.ch) // close output channel
	c.ch = nil          // and remove it
	c.Unlock()
}

func (c *channel) isActive() bool {
	c.Lock()
	defer c.Unlock()

	return c.active
}

func (c *channel) setIsActive(ia bool) {
	c.Lock()
	c.active = ia
	c.Unlock()
}

func (c *channel) out() chan interface{} {
	c.Lock()
	defer c.Unlock()

	return c.ch
}

func closeChanSafe(c chan interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// tried to close c, which was already closed.
		}
	}()

	close(c)
}
