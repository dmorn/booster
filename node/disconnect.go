package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Disconnect performs the steps required to unpair with a remote node.
// laddr is the local booster address to dial with. id is the remote
// node identifier.
func (b *Booster) Disconnect(ctx context.Context, network, laddr, id string) error {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := b.DialContext(_ctx, network, laddr)
	if err != nil {
		return errors.New("booster: unable to contact booster " + laddr + " : " + err.Error())
	}
	defer conn.Close()

	bid, err := hex.DecodeString(id)
	if err != nil {
		return errors.New("booster: " + err.Error())
	}

	buf := make([]byte, 0, 3+len(bid))
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDDisconnect)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, bid...)

	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write disconnect request: " + err.Error())
	}

	buf = make([]byte, len(bid)+4)

	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable to read disconnect response: " + err.Error())
	}

	v := buf[0] // version
	if v != BoosterVersion1 {
		return errors.New("booster: unsupported booster version in disconnect response: " + string(v))
	}

	_ = buf[1]  // cmd
	r := buf[2] // response
	if r != BoosterRespSuccess {
		return errors.New("booster: connect request refused")
	}

	_ = buf[3]  // reserved field
	_ = buf[3:] // id

	return nil
}

func (b *Booster) handleDisconnect(ctx context.Context, conn net.Conn) error {
	buf := make([]byte, 20) // sha1 length
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable to read disconnect request id: " + err.Error())
	}
	id := fmt.Sprintf("%x", buf)

	respWriter := func(err error) error {
		resp := BoosterRespSuccess
		if err != nil {
			resp = BoosterRespGeneralFailure
		}

		buf = make([]byte, 0, len(id)+4)
		buf = append(buf, BoosterVersion1)
		buf = append(buf, BoosterCMDDisconnect)
		buf = append(buf, resp)
		buf = append(buf, BoosterFieldReserved)
		buf = append(buf, id...)

		if _, err := conn.Write(buf); err != nil {
			return errors.New("booster: unable to write connect response: " + err.Error())
		}

		return err
	}

	// do not check if the node is up anymore
	b.Untrace(id)

	// first deactivate the node...
	b.CloseNode(id) // do not check for errors (maybe the node wasy already closed)
	// ...then remove it (this is important)
	if _, err := b.RemoveNode(id); err != nil {
		return respWriter(err)
	}

	return respWriter(nil)
}
