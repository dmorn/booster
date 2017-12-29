package node

import (
	"context"
	"errors"
	"io"
	"net"
)

func (b *Booster) InspectStream(ctx context.Context, network, baddr string, stream chan *RemoteNode, errc chan error) error {
	conn, err := b.DialContext(ctx, network, baddr)
	if err != nil {
		return errors.New("booster: unable to contact node: " + err.Error())
	}

	buf := make([]byte, 0, 3)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDInspect)
	buf = append(buf, BoosterFieldReserved)
	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write inspect request: " + err.Error())
	}

	buf = make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable to read inspect respose: " + err.Error())
	}

	v := buf[0]   // version
	cmd := buf[1] // cmd
	_ = buf[2]    // reserved field
	rsp := buf[3] // response

	if v != BoosterVersion1 {
		return errors.New("booster: unsupported version (" + string(v) + ") in inspect response")
	}

	if cmd != BoosterCMDInspect {
		return errors.New("booster: unexpected cmd: " + string(cmd))
	}

	if rsp != BoosterRespSuccess {
		return errors.New("booster: inspect request failed")
	}

	// stream starts
	if _, err := io.ReadFull(conn, buf[:1]); err != nil {
		return errors.New("booster: unable to read stream start signal " + err.Error())
	}

	m := buf[0]
	if m != BoosterStreamStart {
		return errors.New("booster: stream did not start")
	}

	c := make(chan error)
	go func(conn net.Conn) {
		for {
			// ask for a node
			if _, err := conn.Write([]byte{BoosterStreamNext}); err != nil {
				c <- errors.New("booster: unable to write next message: " + err.Error())
				return
			}

			// check if there is one
			if _, err := io.ReadFull(conn, buf[:1]); err != nil {
				c <- errors.New("booster: unable to read step message: " + err.Error())
				return
			}
			m := buf[0]
			if m != BoosterStreamNext {
				c <- errors.New("booster: stream closed")
				return
			}

			// read the node
			node, err := ReadRemoteNode(conn)
			if err != nil {
				c <- err
				return
			}
			stream <- node
		}
	}(conn)

	go func() {
		select {
		case <-ctx.Done():
			errc <- ctx.Err()
			conn.Close()
		case err := <-c:
			errc <- err
			conn.Close()
		}
	}()

	return nil
}

func (b *Booster) handleInspect(ctx context.Context, conn net.Conn) error {
	buf := make([]byte, 0, 4)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDInspect)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterRespSuccess)
	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write inspect response: " + err.Error())
	}

	// send streaming start signal
	if _, err := conn.Write([]byte{BoosterStreamStart}); err != nil {
		return errors.New("booster: unable to send start stream signal: " + err.Error())
	}

	defer func() {
		//discard step message
		_, _ = io.ReadFull(conn, buf[:1])

		// send streaming stop signal
		_, _ = conn.Write([]byte{BoosterStreamStop})
	}()

	// TODO(daniel): need to keep on sending nodes when their workload (or others) value gets updated.
	stream := b.Sub(TopicRemoteNodes)
	for i := range stream {
		n := i.(*RemoteNode)

		buf = make([]byte, 1)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return errors.New("booster: unable to read stream step message: " + err.Error())
		}

		m := buf[0] // next or stop message
		if m == BoosterStreamStop || m != BoosterStreamNext {
			_, err := conn.Write(buf) // send the stop message back and return
			return err
		}

		// say that we have something to send
		if _, err := conn.Write([]byte{BoosterStreamNext}); err != nil {
			return errors.New("booster: unable to write next message: " + err.Error())
		}

		buf, err := n.EncodeBinary()
		if err != nil {
			return errors.New("booster: unable to encode node: " + err.Error())
		}

		if _, err := conn.Write(buf); err != nil {
			return errors.New("booster: unable to write inspect response: " + err.Error())
		}
	}


	return nil
}
