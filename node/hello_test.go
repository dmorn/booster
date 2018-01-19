package node_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"

	"github.com/danielmorandini/booster-network/node"
)

func TestHello(t *testing.T) {
	ip := net.IPv4(127, 0, 1, 165)
	port := 1099

	mockConn := newConn()
	mockConn.remoteAddr = &net.TCPAddr{
		IP:   ip,
		Port: port,
	}
	d := new(connectDialer)
	d.Conn = mockConn.client

	br := node.NewBoosterDefault()
	br.Dialer = d

	c := make(chan error)
	go func() {
		ctx := context.Background()
		// usually the port here should be the booster listening port.
		// does not matter here. Andrea: I love such comments
		addr := net.JoinHostPort(mockConn.remoteAddr.String(), strconv.Itoa(port))
		_, paddr, err := br.Hello(ctx, "fakenet", addr)
		if err != nil {
			c <- fmt.Errorf("hello error: %v", err)
			return
		}

		if paddr != addr {
			c <- fmt.Errorf("unexpected address. found %v, wanted %v", paddr, addr)
			return
		}

		c <- nil
	}()

	go func(conn net.Conn) {
		buf := make([]byte, 3)
		if _, err := io.ReadFull(conn, buf); err != nil {
			c <- err
			return
		}

		if buf[0] != node.BoosterVersion1 {
			c <- fmt.Errorf("unexpected version %v", buf[0])
			return
		}

		if buf[1] != node.BoosterCMDHello {
			c <- fmt.Errorf("unexpected cmd %v", buf[1])
			return
		}

		_ = buf[2] // reserved field

		// write response back
		buf = make([]byte, 0, 6)
		buf = append(buf, node.BoosterVersion1)
		buf = append(buf, node.BoosterCMDHello)
		buf = append(buf, node.BoosterRespSuccess)
		buf = append(buf, node.BoosterFieldReserved)
		buf = append(buf, byte(port>>8), byte(port))

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
