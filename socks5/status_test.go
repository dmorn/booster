package socks5_test

import (
	"testing"

	"github.com/danielmorandini/booster/socks5"
)

func TestRegisterStatusListener_duplicateID(t *testing.T) {
	s := new(socks5.Socks5)
	c := make(chan<- uint8)
	id := "foo"

	if err := s.RegisterStatusListener(id, c); err != nil {
		t.Fatal(err)
	}

	if err := s.RegisterStatusListener(id, c); err == nil {
		t.Fatal("expected duplicate id error")
	}

}

func TestRemoveStatusListener(t *testing.T) {
	s := new(socks5.Socks5)
	c := make(chan<- uint8)
	id := "foo"

	if err := s.RegisterStatusListener(id, c); err != nil {
		t.Fatal(err)
	}

	s.RemoveStatusListener(id)

	if err := s.RegisterStatusListener(id, c); err != nil {
		t.Fatal(err)
	}
}
