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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/websocket"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/protocol"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [host:port -- optional]",
	Short: "starts an http server sercing ws connections on /monitor. Features to be monitored can be chosen",
	Long:  `todo`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		if len(args) == 1 {
			addr = args[0]
		}

		// registered handlers
		monitor := newMonitor(addr, bport, pport)
		http.Handle("/monitor", websocket.Handler(monitor.handler))

		port := ":4000"
		log.Info.Printf("Listening on port: %v", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Error.Printf("ListenAndServe: %v", err)
		}
	},
}

type monitor struct {
	target  string
	booster *booster.Booster
}

func newMonitor(target string, pp, bp int) *monitor {
	b, err := booster.New(pp, bp)
	if err != nil {
		panic(err)
	}

	return &monitor{
		target:  target,
		booster: b,
	}
}

func (m *monitor) handler(conn *websocket.Conn) {
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// monitor all features, can be filtered later
	features := []protocol.Message{protocol.MessageNode, protocol.MessageBandwidth}
	stream, err := m.booster.Inspect(ctx, "tcp", m.target, features)
	if err != nil {
		log.Error.Printf("handler: %v", err)
		return
	}

	for i := range stream {
		if node, ok := i.(*protocol.PayloadNode); ok {
			if err = handleNode(conn, node); err != nil {
				log.Error.Println(err.Error())
				return
			}
			continue
		}

		if bw, ok := i.(*protocol.PayloadBandwidth); ok {
			if err = handleBandwidth(conn, bw); err != nil {
				log.Error.Println(err.Error())
				return
			}
			continue
		}

		log.Error.Printf("unrecognised message: %+v", i)
	}
}

func handleNode(conn *websocket.Conn, node *protocol.PayloadNode) error {
	log.Debug.Printf("[%v] node received", node.ID)

	if err := SendNode(conn, node); err != nil {
		return fmt.Errorf("[%v] unable to send msg: %v", node.ID, err)
	}

	return nil
}

func handleBandwidth(conn *websocket.Conn, bw *protocol.PayloadBandwidth) error {
	log.Debug.Printf("[%v] bandwidth message received", bw.Type)

	if err := SendBandwidth(conn, bw); err != nil {
		return fmt.Errorf("[%v] unable to send msg: %v", bw.Type, err)
	}

	return nil
}

func SendNode(conn *websocket.Conn, msg *protocol.PayloadNode) error {
	return send(conn, "node", msg)
}

func SendBandwidth(conn *websocket.Conn, msg *protocol.PayloadBandwidth) error {
	return send(conn, "net", msg)
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func send(conn *websocket.Conn, t string, v interface{}) error {
	msg := &Message{
		Type: t,
		Data: v,
	}

	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(msg); err != nil {
		return err
	}

	io.Copy(conn, b)

	return nil
}
