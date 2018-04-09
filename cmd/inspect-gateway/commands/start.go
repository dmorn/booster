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

		features := []protocol.Message{protocol.MessageNode}
		stream, err := b.Inspect(context.Background(), "tcp", addr, features)
		if err != nil {
			fmt.Println(err)
			return
		}

		for i := range stream {
			node, ok := i.(*protocol.PayloadNode)
			if !ok {
				log.Error.Printf("unrecognised message: %+v", i)
				continue
			}

			log.Printf("[%v] node received", node.ID)

			if err = c.Update(node); err != nil {
				log.Fatalf("[%v] unable to send msg: %v", node.ID, err)
			}
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

func (c *client) FetchToken() error {
	url := fmt.Sprintf("http://%v:%v/token", c.host, c.port)
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

func (c *client) Update(msg *protocol.PayloadNode) error {
	url := fmt.Sprintf("http://%v:%v/node?token=%v", c.host, c.port, c.token)
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
