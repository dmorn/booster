/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
				// we tried to send something trough ch and we found it closed.
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
