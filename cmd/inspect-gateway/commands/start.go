package commands

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"
	"io"
	"io/ioutil"
	"net/http"
	"net"
	"encoding/json"

	"github.com/danielmorandini/booster-network/node"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command {
	Use: "start",
	Short: "inspects the node's activity and sends the data to the target API",
	Long: ``,
	Args: cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		host, port, err := net.SplitHostPort(targetAddr)
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
		b := node.NewBoosterDefault()
		stream := make(chan *node.Node)
		errc := make(chan error)
		ctx := context.Background()

		err = b.InspectStream(ctx, "tcp", boosterAddr, stream, errc)
		if err != nil {
			log.Fatalf("unable to inspect node @ [%v]: %v", boosterAddr, err)
		}

		for {
			select {
			case err := <-errc:
				log.Fatalf("inspect error: %v", err)
			case node := <-stream:
				log.Println("[%v] msg received", node.ID())
				msg := newMsg(node)

				if err = c.Update(msg); err != nil {
					log.Fatalf("[%v] unable to send msg: %v", msg.ID, err)
				}
			}
		}
	},
}

type nodeMsg struct {
	ID string `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	BAddr string `json:"booster_address"`
	PAddr string `json:"proxy_address"`
	Workload int `json:"workload"`
	LastOp *operation `json:"last_operation"`
}

type operation struct {
	ID string `json:"id"`
	Code int `json:"code"`
}

func newMsg(n *node.Node) *nodeMsg {
	return &nodeMsg {
		ID: n.ID(),
		BAddr: n.BAddr.String(),
		PAddr: n.PAddr.String(),
		Workload: n.Workload(),
		LastOp: &operation {
			ID: n.LastOperation().ID,
			Code: int(n.LastOperation().Op),
		},
	}
}

type client struct {
	*http.Client

	host string
	port string
	token string
}

func newAPIClient(host, port string) *client {
	return &client{
		host: host,
		port: port,
		Client: &http.Client {
			Timeout: time.Second*5,
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
	if err := json.NewDecoder(resp.Body).Decode( &struct {
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

func (c *client) Update(msg *nodeMsg) error {
	url := fmt.Sprintf("http://%v:%v/node/%v", c.host, c.port, c.token)
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
