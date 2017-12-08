package node_test

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/danielmorandini/booster-network/node"
	"github.com/danielmorandini/booster-network/socks5"
)

type connectDialer struct {
	net.Conn
}

func (c *connectDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c, nil
}

func TestConnect(t *testing.T) {
	mockConn := newConn()
	d := new(connectDialer)
	d.Conn = mockConn.client

	br := node.BOOSTER()
	br.Dialer = d

	h := sha1.New()
	h.Write([]byte("foo"))

	trgID := fmt.Sprintf("%x", h.Sum(nil)) // traget ID that will be checked against the return value of connect
	raddr := "127.0.1.65:66"               // fake remote booster address

	c := make(chan error)
	go func() {
		ctx := context.Background()
		id, err := br.Connect(ctx, "fakenet", "fakeladdr", raddr)
		if err != nil {
			c <- err
			return
		}

		if id != trgID {
			c <- err
			return
		}

		c <- nil
	}()

	go func(conn net.Conn) {
		// we have to act as if we were the remote booster instance answering
		buf := make([]byte, 3)
		if _, err := io.ReadFull(conn, buf); err != nil {
			c <- err
			return
		}

		if buf[0] != node.BoosterVersion1 {
			c <- fmt.Errorf("unexpected version: %v", buf[0])
			return
		}

		if buf[1] != node.BoosterCMDConnect {
			c <- fmt.Errorf("unexpencted CMD response: %v", buf[1])
			return
		}
		_ = buf[2] // reserved field

		addr, err := socks5.ReadAddress(conn)
		if err != nil {
			c <- err
		}

		// parse IP that we collected from connection
		// and the one that we sent and compare them
		ip, port, err := net.SplitHostPort(addr)
		if err != nil {
			c <- err
			return
		}
		trgIP, trgPort, err := net.SplitHostPort(raddr)
		if err != nil {
			c <- err
			return
		}

		if ip != trgIP {
			c <- fmt.Errorf("unexpected IP. found %v, wanted %v", ip, trgIP)
			return
		}

		if port != trgPort {
			c <- fmt.Errorf("unexpected port. found %v, wanted %v", port, trgPort)
			return
		}

		// at this point we should make a hello request to the remote server,
		// collect the reponse, add the remote booster as a node, create a sha1
		// id of the node and send it back. We'll skip and check only the last
		// step here.
		bid, err := hex.DecodeString(trgID)
		if err != nil {
			c <- err
			return
		}

		buf = make([]byte, 0, len(bid)+4)
		buf = append(buf, node.BoosterVersion1)
		buf = append(buf, node.BoosterCMDConnect)
		buf = append(buf, node.BoosterRespSuccess)
		buf = append(buf, node.BoosterFieldReserved)
		buf = append(buf, bid...)

		if _, err := conn.Write(buf); err != nil {
			c <- err
			return
		}

		c <- nil
	}(mockConn.server)

	if s := <-c; s != nil {
		t.Fatal(s)
	}
}
