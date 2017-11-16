package socks5_test

import (
	"testing"

	"github.com/danielmorandini/booster/socks5"
)

func TestRegisterWorkloadListener_duplicateID(t *testing.T) {
	s := new(socks5.Socks5)
	c := make(chan<- int)
	id := "foo"

	if err := s.RegisterWorkloadListener(id, c); err != nil {
		t.Fatal(err)
	}

	if err := s.RegisterWorkloadListener(id, c); err == nil {
		t.Fatal("expected duplicate id error")
	}

}

func TestRemoveWorkloadListener(t *testing.T) {
	s := new(socks5.Socks5)
	c := make(chan<- int)
	id := "foo"

	if err := s.RegisterWorkloadListener(id, c); err != nil {
		t.Fatal(err)
	}

	s.RemoveWorkloadListener(id)

	if err := s.RegisterWorkloadListener(id, c); err != nil {
		t.Fatal(err)
	}
}
