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

package node_test

import (
	"testing"

	"github.com/danielmorandini/booster/node"
)

func TestNewNode(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("%v, wanted %v", n.Workload(), 0)
	}

	if !n.IsLocal() {
		t.Fatalf("%v, wanted true. Node is not local", n.IsLocal())
	}

	if n.IsActive() {
		t.Fatalf("node is active")
	}

	n.SetIsActive(true)
	if !n.IsActive() {
		t.Fatalf("node is NOT active")
	}
}

func TestAddTunnel(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	tunnel := node.NewTunnel("host:8888")
	n.AddTunnel(tunnel)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	n.AddTunnel(tunnel)

	if n.Workload() != 2 {
		t.Fatalf("workload: %v, wanted 2", n.Workload())
	}
}

func TestRemoveTunnel(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	tunnel := node.NewTunnel("host:8888")
	n.AddTunnel(tunnel)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	n.AddTunnel(tunnel)

	if n.Workload() != 2 {
		t.Fatalf("workload: %v, wanted 2", n.Workload())
	}

	if err := n.RemoveTunnel(tunnel.Target, false); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	if err := n.RemoveTunnel(tunnel.Target, false); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("workload: %v, wanted 0", n.Workload())
	}

	if err := n.RemoveTunnel(tunnel.Target, false); err == nil {
		t.Fatal("err should not be nil")
	}
}

