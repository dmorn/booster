package node

import (
	"context"
	"errors"
	"io"
	"net"
)

func (b *Booster) InspectSub(ctx context.Context, network, baddr string, nodec chan *RemoteNode) error {
	conn, err := b.DialContext(ctx, network, baddr)
	if err != nil {
		return errors.New("booster: unable to contact node: " + err.Error())
	}
	defer conn.Close()

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

	errc := make(chan error)
	go func() {
		for {
			node, err := ReadRemoteNode(conn)
			if err != nil {
				errc <- err
				return
			}
			nodec <- node
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func (b *Booster) handleInspect(ctx context.Context, conn net.Conn) error {
	buf := make([]byte, 3)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable to read inspect request: " + err.Error())
	}

	v := buf[0]   // version
	cmd := buf[1] // command
	_ = buf[2]    // reserved field

	if v != BoosterVersion1 {
		return errors.New("booster: unsupported version " + string(v))
	}

	if cmd != BoosterCMDInspect {
		return errors.New("booster: unexpected command: " + string(cmd))
	}

	buf = make([]byte, 0, 4)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDInspect)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterRespSuccess)
	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write inspect response: " + err.Error())
	}

	// TODO(daniel): nedd to keep on sending nodes when their workload (or others) value gets updated.
	for _, n := range b.GetNodes() {
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
