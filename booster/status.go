package booster

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/danielmorandini/booster-network/socks5"
)

// ServeStatus writes the proxy's status to the connection, whenever it changes.
func (b *Booster) ServeStatus(ctx context.Context, conn net.Conn) error {
	ec := make(chan error)
	wc, err := b.Proxy.Sub(socks5.TopicWorkload)
	if err != nil {
		return err
	}

	// send status messages.
	go func() {
		defer func() {
			b.Proxy.Unsub(wc, socks5.TopicWorkload)
		}()

		buf := make([]byte, 0, 4+32)
		buf = append(buf, BoosterVersion1)
		buf = append(buf, BoosterCMDStatus)
		buf = append(buf, BoosterFieldReserved)
		for i := range wc {
			wm, ok := i.(socks5.WorkloadMessage)
			if !ok {
				ec <- errors.New("booster: unable to recognise workload message")
				return
			}

			abuf, err := socks5.EncodeAddressBinary(wm.Target)
			if err != nil {
				ec <- errors.New("booster: unable to encode target: " + err.Error())
				return
			}

			buf = buf[:3]
			buf = append(buf, byte(wm.Event))
			buf = append(buf, abuf...)

			if _, err := conn.Write(buf); err != nil {
				ec <- errors.New("booster: unable to write status: " + err.Error())
			}
		}
	}()

	select {
	case <-ctx.Done():
		b.Proxy.Unsub(wc, socks5.TopicWorkload)
		return errors.New("booster: serve status: " + ctx.Err().Error())
	case err := <-ec:
		b.Proxy.Unsub(wc, socks5.TopicWorkload)
		return err
	}
}

// Status expects conn to produce booster status messages. It then
// uses the data received to update the workload's value of the node.
//
// If the connection is closed, the data is somehow corrupted or a cancel
// signal is received, it closes the connection and sets the IsActive value
// of the node to false.
//
// Publishes a TopicNodes message when a node is updated.
func (b *Booster) Status(ctx context.Context, node Node) error {
	ctx, cancel := context.WithCancel(ctx)

	conn, err := b.DialContext(ctx, node.Addr().Network(), node.Addr().String())
	if err != nil {
		cancel()
		return errors.New("status error: " + err.Error())
	}

	buf := make([]byte, 0, 3)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDStatus)
	buf = append(buf, BoosterFieldReserved)
	if _, err := conn.Write(buf); err != nil {
		cancel()
		return errors.New("unable to write status request: " + err.Error())
	}

	node.SetIsActive(true)

	fail := func(err error) {
		b.Printf("booster: status error: %v", err)

		conn.Close()
		cancel()
	}

	buf = make([]byte, 4)
	errc := make(chan error)

	// keep on reading status messages until the node is closed.
	go func() {
		for {
			// check if the node is active before trying to read from the connection
			if !node.IsActive() {
				return
			}

			if _, err := io.ReadFull(conn, buf); err != nil {
				errc <- err
				continue
			}

			_ = buf[0]                           // version - already checked in the hello procedure
			_ = buf[1]                           // command
			_ = buf[2]                           // reserved field
			event := int(buf[3])                       // event

			target, err := socks5.ReadAddress(conn)
			if err != nil {
				errc <- errors.New("booster: unable to read target: " + err.Error())
				continue
			}

			addr, err := net.ResolveTCPAddr("tcp", target)
			if err != nil {
				errc <- errors.New("node: unable to create addr: " + err.Error())
				continue
			}

			switch event {
			case socks5.EventPush:
				b.AddTunnel(node, addr)
			case socks5.EventPop:
				if err = b.Ack(node, addr); err != nil {
					errc <- err
				}

			default:
				errc <- errors.New("booster: unrecognised event: " + strconv.Itoa(event))
			}

		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			fail(err)
			return
		case err := <-errc:
			b.Printf("booster: status error: %v", err)
		case <-node.Stop():
			fail(errors.New("stopped"))
			return
		}
	}()

	return nil
}
