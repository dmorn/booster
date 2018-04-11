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
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/protocol"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [host:port -- optional]",
	Short: "inspects the node's activity and sends the data to the target API",
	Long:  `start inspects (by default) the local node listening @ localhost:4884, takes the messages received and sends them to the apiEndpoint, json encoded. An optional booster address could be passed as paramenter.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		if len(args) == 1 {
			addr = args[0]
		}

		host, port, err := net.SplitHostPort(apiEndpoint)
		if err != nil {
			log.Fatalf("unable to split host port: %v", err)
		}

		c := newAPIClient(host, port)

		// first get an identification token
		log.Print("fetching token...")
		if err = c.FetchToken(); err != nil {
			log.Fatalf("unable to fetch token: %v", err)
		}
		log.Printf("token: %v", c.token)

		// listen for node events
		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		features := []protocol.Message{protocol.MessageNode, protocol.MessageBandwidth}
		stream, err := b.Inspect(context.Background(), "tcp", addr, features)
		if err != nil {
			fmt.Println(err)
			return
		}

		for i := range stream {
			if node, ok := i.(*protocol.PayloadNode); ok {
				if err = c.handleNode(node); err != nil {
					log.Info.Println(err.Error())
				}

				continue
			}

			if bw, ok := i.(*protocol.PayloadBandwidth); ok {
				if err = c.handleBandwidth(bw); err != nil {
					log.Info.Println(err.Error())
				}
				continue
			}

			log.Error.Printf("unrecognised message: %+v", i)
		}
	},
}

type client struct {
	*http.Client

	host  string
	port  string
	token string
}

func newAPIClient(host, port string) *client {
	return &client{
		host: host,
		port: port,
		Client: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

func (c *client) api() string {
	return fmt.Sprintf("http://%v:%v/api", c.host, c.port)
}

func (c *client) FetchToken() error {
	url := fmt.Sprintf("%v/token", c.api())
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var token string
	if err := json.NewDecoder(resp.Body).Decode(&struct {
		Token *string `json:"token"`
	}{&token}); err != nil {
		return err
	}

	if token == "" {
		return fmt.Errorf("token is empty")
	}

	c.token = token
	return nil
}

func (c *client) handleNode(node *protocol.PayloadNode) error {
	log.Debug.Printf("[%v] node received", node.ID)

	if err := c.UpdateNode(node); err != nil {
		return fmt.Errorf("[%v] unable to send msg: %v", node.ID, err)
	}

	return nil
}

func (c *client) handleBandwidth(bw *protocol.PayloadBandwidth) error {
	log.Debug.Printf("[%v] bandwidth message received", bw.Type)

	if err := c.UpdateBandwidth(bw); err != nil {
		return fmt.Errorf("[%v] unable to send msg: %v", bw.Type, err)
	}

	return nil
}

func (c *client) UpdateNode(msg *protocol.PayloadNode) error {
	url := fmt.Sprintf("%v/node?token=%v", c.api(), c.token)
	return c.update(url, msg)
}

func (c *client) UpdateBandwidth(msg *protocol.PayloadBandwidth) error {
	url := fmt.Sprintf("%v/bandwidth?token=%v", c.api(), c.token)

	return c.update(url, msg)
}

func (c *client) update(url string, msg interface{}) error {
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(msg); err != nil {
		return err
	}

	resp, err := c.Post(url, "application/json", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	return nil
}
