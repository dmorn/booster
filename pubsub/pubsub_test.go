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

package pubsub_test

import (
	"testing"
	"time"

	"github.com/danielmorandini/booster/pubsub"
)

func TestSub(t *testing.T) {
	ps := pubsub.New()
	if _, err := ps.Sub("t1", nil); err != nil {
		t.Fatal(err)
	}

	if err := ps.Close("t1"); err != nil {
		t.Fatal(err)
	}
}

func TestPub(t *testing.T) {
	ps := pubsub.New()
	wait := make(chan struct{})
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1 not responding")
	})

	if _, err := ps.Sub("t1", func(i interface{}) {
		if i != "fakedata" {
			t.Fatalf("unexpected data: %v", i)
		}

		go func() {
			wait <- struct{}{}
		}()
	}); err != nil {
		t.Fatal(err)
	}

	ps.Pub("fakedata", "t1")

	<-wait
	timer.Stop()
}

func TestPub_multiple(t *testing.T) {
	ps := pubsub.New()
	wait := make(chan struct{}, 2)
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1/t2 not responding")
	})

	if _, err := ps.Sub("t1", func(d interface{}) {
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}

		go func() {
			wait <- struct{}{}
		}()
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := ps.Sub("t2", func(d interface{}) {
		if d != "fakedata" {
			t.Fatalf("unexpected data: %v", d)
		}

		go func() {
			wait <- struct{}{}
		}()
	}); err != nil {
		t.Fatal(err)
	}

	ps.Pub("fakedata", "t1")
	ps.Pub("fakedata", "t2")

	<-wait
	<-wait
	timer.Stop()
}

func TestUnsub(t *testing.T) {
	ps := pubsub.New()
	index, err := ps.Sub("t1", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := ps.Unsub(index, "t1"); err != nil {
		t.Fatal(err)
	}
}

func TestClose(t *testing.T) {
	ps := pubsub.New()
	if _, err := ps.Sub("t1", nil); err != nil {
		t.Fatal(err)
	}

	if err := ps.Close("t1"); err != nil {
		t.Fatal(err)
	}
}

func TestMultiSub_concurrent(t *testing.T) {
	ps := pubsub.New()
	wait := make(chan struct{}, 4)
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1/t2 not responding")
	})

	// these two messages shuold be ignored
	ps.Pub("fake_data_t1", "t1")
	ps.Pub("fake_data_t2", "t2")

	var err1, err2 error
	var it1, it2 int

	go func() {
		if it1, err1 = ps.Sub("t1", func(d interface{}) {
			if d != "foo" {
				t.Fatalf("unexpected data from t1: %v", d)
			}

			wait <- struct{}{}
		}); err1 != nil {
			t.Fatal(err1)
		}

		wait <- struct{}{}
	}()

	go func() {
		if it2, err2 = ps.Sub("t2", func(d interface{}) {
			if d != "bar" {
				t.Fatalf("unexpected data from t2: %v", d)
			}

			wait <- struct{}{}
		}); err2 != nil {
			t.Fatal(err2)
		}

		wait <- struct{}{}
	}()

	// wait for subscription
	<-wait
	<-wait

	ps.Pub("foo", "t1")
	ps.Pub("bar", "t2")

	// wait for the functions to be called
	<-wait
	<-wait
	timer.Stop()

	if err := ps.Unsub(it1, "t1"); err != nil {
		t.Fatalf("unable to unsub ch1 from t1: %v", err)
	}
	if err := ps.Unsub(it2, "t2"); err != nil {
		t.Fatalf("unable to unsub ch1 from t2: %v", err)
	}

}

func TestMultiSub(t *testing.T) {
	ps := pubsub.New()

	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1 not responding")
	})
	wait := make(chan struct{}, 2)

	f := func() {
		if _, err := ps.Sub("t1", func(i interface{}) {
			wait <- struct{}{}
		}); err != nil {
			t.Fatal(err)
		}
	}

	f()
	f()

	ps.Pub("hi", "t1")

	<-wait
	<-wait
	timer.Stop()
}
