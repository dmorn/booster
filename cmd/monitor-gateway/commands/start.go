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

package commands

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [x]",
	Short: "starts an http server serving ws connections on /monitor. Connections serve information about node x in JSON encoded real-time messages.",
	Long: `starts an http server serving ws connections on /monitor. Connections serve information about node x in JSON encoded real-time messages.
	
x: node address (format host:port) that will be monitored (optional, default localhost:4884)

	Example:
	bin/monitor-gateway start
	2018/05/28 17:47:23.247894 Listening on port: :4000

Features allowed: node | net
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		if len(args) == 1 {
			addr = args[0]
		}

		// registered handlers
		monitor := newMonitor(feature, addr, bport, pport)
		http.HandleFunc("/monitor", monitor.handler)

		port := ":4000"
		log.Info.Printf("listening on port: %v", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Error.Printf("ListenAndServe: %v", err)
		}
	},
}

type monitor struct {
	target  string
	feature string
	booster *booster.Booster
}

func newMonitor(t, f string, pp, bp int) *monitor {
	b, err := booster.New(pp, bp)
	if err != nil {
		panic(err)
	}

	return &monitor{
		target:  t,
		feature: f,
		booster: b,
	}
}

func (m *monitor) handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error.Println(err)
		return
	}
	defer conn.Close()

	fail := func() {
		log.Info.Println("closing handler...")
		conn.Close()
		cancel()
	}

	// keep on reading pong messages
	go func() {
		pongWait := time.Second * 20
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		for {
			if _, _, err := conn.NextReader(); err != nil {
				fail()
				break
			}
		}
	}()

	var feature protocol.MonitorFeature
	switch m.feature {
	case "node":
		feature = protocol.MonitorNode
	case "net":
		feature = protocol.MonitorNet
	}

	var wg sync.WaitGroup
	wg.Add(1)
	if err := m.booster.Monitor(ctx, "tcp", m.target, booster.Inspection{
		Feature: feature,
		Run: func(module packet.Module) error {
			return send(conn, m.feature, module.Payload())
		},
		PostRun: func(err error) {
			log.Error.Printf("handler: %v", err)
			fail()
			wg.Done()
		},
	}); err != nil {
		log.Error.Printf("handler: %v", err)
		fail()
		wg.Done()
	}

	ticker := time.NewTicker(time.Second * 4)
	defer ticker.Stop()

	go func() {
		for _ = range ticker.C {
			writeWait := time.Second * 2
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error.Printf("handler: %v", err)
				fail()
				wg.Done()
			}
		}
	}()

	wg.Wait()
}

type Message struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func send(conn *websocket.Conn, t string, p []byte) error {
	msg := &Message{
		Type: t,
		Data: p,
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
	websocket.WriteJSON(conn, msg)

	return nil
}
