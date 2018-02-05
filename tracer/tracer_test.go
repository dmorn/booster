package tracer_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/danielmorandini/booster-network/tracer"
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
	tr := tracer.NewDefault()
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
	tr := tracer.NewDefault()
	p := &pg{shouldFail: false, id: "fake"} // it looks like the host is up

	if err := tr.Trace(p); err != nil {
		t.Fatal(err)
	}

	stream := tr.Sub(tracer.TopicConnDiscovered)
	defer func() {
		tr.Unsub(stream, tracer.TopicConnDiscovered)
	}()

	i := <-stream
	m, ok := i.(tracer.Message)
	if !ok {
		t.Fatalf("wrong trace data found: %v", i)
	}

	if m.ID != "fake" {
		t.Fatalf("found wrong id: wanted %v, found %v", "fake", m.ID)
	}
}
