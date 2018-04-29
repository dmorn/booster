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

package tracer_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/danielmorandini/booster/pubsub"
	"github.com/danielmorandini/booster/tracer"
)

type pg struct {
	id         string
	shouldFail bool
}

type addr struct {
}

func (p *pg) Addr() net.Addr {
	return new(addr)
}

func (a *addr) String() string {
	return "host:port"
}

func (a *addr) Network() string {
	return "tcp"
}

func (p *pg) ID() string {
	return p.id
}

func (p *pg) Ping(ctx context.Context) error {
	if p.shouldFail {
		return errors.New("should fail")
	}

	return nil
}

func TestRun(t *testing.T) {
	tr := tracer.New()

	if err := tr.Run(); err != nil {
		t.Fatal(err)
	}
	if tr.Status() != tracer.StatusRunning {
		t.Fatalf("unexpected tracer status: found %v, expected %v", tr.Status(), tracer.StatusRunning)
	}

	if err := tr.Run(); err == nil {
		t.Fatal("tracer should be running")
	}

	tr.Close()
	if tr.Status() != tracer.StatusStopped {
		t.Fatalf("unexpected tracer status: found %v, expected %v", tr.Status(), tracer.StatusStopped)
	}
}

func TestTrace(t *testing.T) {
	tr := tracer.New()
	tr.RefreshRate = time.Millisecond

	if err := tr.Run(); err != nil {
		t.Fatal(err)
	}
	p := &pg{shouldFail: false, id: "fake"} // it looks like the host is up

	if err := tr.Trace(p); err != nil {
		t.Fatal(err)
	}

	wait := make(chan struct{}, 1)
	cancel, err := tr.Sub(&pubsub.Command{
		Topic: tracer.TopicConn,
		Run: func(i interface{}) error {
			m, ok := i.(tracer.Message)
			if !ok {
				t.Fatalf("wrong trace data found: %v", i)
			}

			if m.ID != "fake" {
				t.Fatalf("found wrong id: wanted %v, found %v", "fake", m.ID)
			}

			wait <- struct{}{}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer cancel()

	<-wait
}
