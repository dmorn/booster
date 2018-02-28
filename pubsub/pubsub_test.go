package pubsub_test

import (
	"testing"
	"time"

	"github.com/danielmorandini/booster/pubsub"
)

func TestSub(t *testing.T) {
	ps := pubsub.New()
	ch1, err := ps.Sub("t1")
	if err != nil {
		t.Fatal(err)
	}
	if ch1 == nil {
		t.Fatal("channel not created")
	}

	if err := ps.Close("t1"); err != nil {
		t.Fatal(err)
	}
}

func TestPub(t *testing.T) {
	ps := pubsub.New()
	ch1, err := ps.Sub("t1")
	if err != nil {
		t.Fatal(err)
	}

	ps.Pub("fakedata", "t1")

	select {
	case d := <-ch1:
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}
	case <-time.After(time.Second * 1):
		t.Fatal("cannot read from ch1")
	}
}

func TestPub_multiple(t *testing.T) {
	ps := pubsub.New()
	ch1, _ := ps.Sub("t1")
	ch2, _ := ps.Sub("t2")

	ps.Pub("fakedata", "t1")
	ps.Pub("fakedata", "t2")

	select {
	case d := <-ch1:
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}
	case <-time.After(time.Second * 1):
		t.Fatal("cannot read from ch1")
	}

	select {
	case d := <-ch2:
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}
	case <-time.After(time.Second * 1):
		t.Fatal("cannot read from ch2")
	}
}

func TestUnsub(t *testing.T) {
	ps := pubsub.New()
	ch1, _ := ps.Sub("t1")

	if err := ps.Unsub(ch1, "t1"); err != nil {
		t.Fatal(err)
	}

	if _, ok := <-ch1; ok {
		t.Fatal("channel is not closed")
	}
}

func TestClose(t *testing.T) {
	ps := pubsub.New()
	ch1, _ := ps.Sub("t1")

	if err := ps.Close("t1"); err != nil {
		t.Fatal(err)
	}

	if _, ok := <-ch1; ok {
		t.Fatal("channel is not closed")
	}
}

func TestMultiSub_concurrent(t *testing.T) {
	ps := pubsub.New()

	// these two messages shuold be ignored
	ps.Pub("fake_data_t1", "t1")
	ps.Pub("fake_data_t2", "t2")

	var ch1 chan interface{}
	var ch2 chan interface{}
	wait := make(chan struct{}, 2)

	go func() {
		ch1, _ = ps.Sub("t1")
		wait <- struct{}{}
	}()

	go func() {
		ch2, _ = ps.Sub("t2")
		wait <- struct{}{}
	}()

	<-wait
	<-wait
	ps.Pub("foo", "t1")
	ps.Pub("bar", "t2")

	select {
	case d := <-ch1:
		if d != "foo" {
			t.Fatalf("unexpected data from ch1: %v", d)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("counld read from ch1")
	}

	select {
	case d := <-ch2:
		if d != "bar" {
			t.Fatalf("unexpected data from ch2: %v", d)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("counld read from ch2")
	}

	if err := ps.Unsub(ch1, "t1"); err != nil {
		t.Fatalf("unable to unsub ch1 from t1: %v", err)
	}
	if err := ps.Unsub(ch2, "t2"); err != nil {
		t.Fatalf("unable to unsub ch1 from t2: %v", err)
	}

}

func TestMultiSub(t *testing.T) {
	// TODO(daniel): remove this skip when the pubsub is fixed. Sometimes this produces a deadlock.
	t.Skip()

	ps := pubsub.New()

	var ch1 chan interface{}
	var ch2 chan interface{}
	var err1 error
	var err2 error
	wait := make(chan struct{}, 2)

	go func() {
		ch1, err1 = ps.Sub("t1")
		if err1 != nil {
			t.Fatal(err1)
		}
		wait <- struct{}{}
	}()

	go func() {
		ch2, err2 = ps.Sub("t1")
		if err2 != nil {
			t.Fatal(err2)
		}
		wait <- struct{}{}
	}()

	<-wait
	<-wait
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
