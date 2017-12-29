package pubsub_test

import (
	"testing"
	"time"

	"github.com/danielmorandini/booster-network/pubsub"
)

func TestSub(t *testing.T) {
	ps := pubsub.New()
	ch1 := ps.Sub("t1")
	if ch1 == nil {
		t.Fatal("channel not created")
	}

	if err := ps.Close("t1"); err != nil {
		t.Fatal(err)
	}
}

func TestLinks(t *testing.T) {
	ps := pubsub.New()
	top := "t1"
	_ = ps.Sub(top)
	c := ps.Sub(top)

	links, err := ps.Links(top)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 2 {
		t.Fatalf("unexpected links length: wanted 2, found %v", len(links))
	}

	ps.Unsub(c, top)
	links, err = ps.Links(top)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 {
		t.Fatalf("unexpected links length: wanted 1, found %v", len(links))
	}
}

func TestPubSub(t *testing.T) {
	ps := pubsub.New()
	ch1 := ps.Sub("t1")
	ch2 := ps.Sub("t2")

	ps.Pub("fakedata", "t1")
	ps.Pub("fakedata", "t2")

	select {
	case d := <-ch1:
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}
	case <- time.After(time.Second * 1):
		t.Fatal("cannot read from ch1")
	}

	select {
	case d := <-ch2:
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}
	case <- time.After(time.Second * 1):
		t.Fatal("cannot read from ch2")
	}

	if err := ps.Close("t1"); err != nil {
		t.Fatal(err)
	}

	ch3 := ps.Sub("t2")
	ps.Pub("fakedata", "t2")

	// ch2 and ch3 should have received the message
	if err := ps.Unsub(ch3, "t2"); err != nil {
		t.Fatal(err)
	}

	select {
	case d := <-ch2:
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}
	case <- time.After(time.Second * 1):
		t.Fatal("cannot read from ch3")
	}

	if _, ok := <-ch3; ok {
		t.Fatal("channel is not closed")
	}
}

func TestUnsub(t *testing.T) {
	ps := pubsub.New()
	ch1 := ps.Sub("t1")

	if err := ps.Unsub(ch1, "t1"); err != nil {
		t.Fatal(err)
	}

	if _, ok := <-ch1; ok {
		t.Fatal("channel is not closed")
	}
}

func TestMultiSub(t *testing.T) {
	ps := pubsub.New()
	ch1 := ps.Sub("t1")
	ch2 := ps.Sub("t1")

	ps.Pub("hi", "t1")

	_, ok := <-ch1
	if !ok {
		t.Fatal("could not read from ch1")
	}
	_, ok = <-ch2
	if !ok {
		t.Fatal("could not read from ch1")
	}
}
