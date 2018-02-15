package pubsub

import (
	"sync"
)

type channel struct {
	sendc chan interface{}
	stopc chan interface{}

	sync.Mutex
	active bool
	ch     chan interface{}
}

func newChannel() *channel {
	ch := &channel{
		sendc:  make(chan interface{}),
		stopc:  make(chan interface{}),
		ch:     make(chan interface{}),
		active: true,
	}

	go func() {
		send := func(m interface{}) {
			ch.out() <- m
		}

		stop := func() {
			ch.setIsActive(false)
			closeChanSafe(ch.sendc)
			closeChanSafe(ch.stopc)
			closeChanSafe(ch.out())
		}

		for {
			select {
			case m := <-ch.sendc:
				send(m)
			case <-ch.stopc:
				stop()
				return
			}
		}
	}()

	return ch
}

func (c *channel) send(m interface{}) {
	if c.isActive() {
		c.sendc <- m
	}
}

func (c *channel) stop() {
	if c.isActive() {
		c.stopc <- struct{}{}
	}
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
