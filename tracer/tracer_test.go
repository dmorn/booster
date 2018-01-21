package tracer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/danielmorandini/booster-network/tracer"
)


type pg struct {
	id         string
	shouldFail bool
}

func (p *pg) String() string {
	return "host:port"
}

func (p *pg) Network() string {
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
	if tr.Status() != tracer.TrackerStatusRunning {
		t.Fatalf("unexpected tracer status: found %v, expected %v", tr.Status(), tracer.TrackerStatusRunning)
	}

	if err := tr.Run(); err == nil {
		t.Fatal("tracer should be running")
	}

	tr.Close()
	if tr.Status() != tracer.TrackerStatusStopped {
		t.Fatalf("unexpected tracer status: found %v, expected %v", tr.Status(), tracer.TrackerStatusStopped)
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
	id, ok := i.(string)
	if !ok {
		t.Fatalf("wrong trace data found: %v", i)
	}

	if id != "fake" {
		t.Fatalf("found wrong id: wanted %v, found %v", "fake", id)
	}
}

