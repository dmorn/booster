package node

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"io"
	"net"
	"time"
)

// Ping sends messages to check if the connection is still alive. Fails if
// it's not able to send after two seconds.
func (b *Booster) Ping(ctx context.Context, node *Node) error {
	conn, err := b.DialContext(ctx, node.Addr().Network(), node.Addr().String())
	if err != nil {
		return errors.New("booster: ping error: " + err.Error())
	}

	var mux sync.Mutex

	fail := func(err error, remove bool) error {
		mux.Lock()
		defer mux.Unlock()

		// be sure that the connection gets closed.
		conn.Close()

		// Do not fail multiple times.
		if !node.IsActive {
			return err
		}

		b.Printf("booster: ping error: %v", err)
		if remove {
			b.CloseNode(node.ID())
			b.RemoveNode(node.ID())
			return err
		} else {
			b.Trace(node)
			b.CloseNode(node.ID())
		}

		return err
	}

	// this function will be fired after NodeIdleTimeout
	timer := time.AfterFunc(b.NodeIdleTimeout, func() {
		fail(errors.New("node flagged as idle"), false)
	})

	errc := make(chan error)

	go func() {
		buf := make([]byte, 0, 3)
		buf = append(buf, BoosterVersion1)
		buf = append(buf, BoosterCMDHeartbeat)
		buf = append(buf, BoosterFieldReserved)

		if _, err := conn.Write(buf); err != nil {
			err = errors.New("booster: unable to send ping message: " + err.Error())
			errc <- fail(err, false)
		}

		buf = buf[:1]
		for {
			errc <- nil // reset timer
			if _, err := io.ReadFull(conn, buf); err != nil {
				err = errors.New("booster: unable to read pong response: " + err.Error())
				errc <- fail(err, false)
			}

			cmd := buf[0] // cmd
			if cmd != BoosterFieldPing {
				err = errors.New("booster: wrong pong response: " + strconv.Itoa(int(cmd)))
				errc <- fail(err, true)
			}

			buf[0] = BoosterFieldPing

			errc <- nil // reset timer
			if _, err := conn.Write(buf); err != nil {
				err = errors.New("booster: unable ping: " + err.Error())
				errc <- fail(err, false)
			}

			// stop the timer while we wait to send another message.
			if !timer.Stop() {
				// already expired
				<-timer.C
				return
			}

			<- time.After(2 * time.Second) // Wait 2 seconds before sending another message.
			errc <- nil // reset timer
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				fail(ctx.Err(), true)
				return
			case err := <-errc:
				if err != nil {
					return
				}
				// Reset the timer if no errors occurred.
				timer.Reset(b.NodeIdleTimeout)
			}
		}
	}()

	return nil
}


func (b *Booster) handlePing(ctx context.Context, conn net.Conn) error {
	errc := make(chan error)

	go func() {
		buf := make([]byte, 0, 1)
		for {
			buf = buf[:1]
			buf = append(buf, BoosterFieldPing)
			if _, err := conn.Write(buf); err != nil {
				errc <- errors.New("booster: unable to write pong message: " + err.Error())
				return
			}

			if _, err := io.ReadFull(conn, buf); err != nil {
				errc <- errors.New("booster: unable to read ping: " + err.Error())
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return errors.New("booster: ping error: " + ctx.Err().Error())
	case err := <-errc:
		return err
	}
}
