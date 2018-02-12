package node

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

// Hello dials with the remote address, expecting it to be a booster server.
// Right after having enstablished the connection, it performs a "Hello" request.
//
// If the response is successfull, it reads the remote proxy address from the response
// and returns it.
func (b *Booster) Hello(ctx context.Context, network, addr string) (string, error) {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := b.DialContext(_ctx, network, addr)
	if err != nil {
		return "", errors.New("booster: unable to contact remote instance: " + err.Error())
	}
	defer conn.Close()

	buf := make([]byte, 0, 3)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDHello)
	buf = append(buf, BoosterFieldReserved)

	if _, err := conn.Write(buf); err != nil {
		return "", errors.New("booster: unable to perform hello request: " + err.Error())
	}

	buf = make([]byte, 6)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", errors.New("booster: unable read hello response: " + err.Error())
	}

	v := buf[0] // version
	if v != BoosterVersion1 {
		return "", errors.New("booster: unsupported version " + strconv.Itoa(int(v)))
	}

	_ = buf[1]                           // cmd
	resp := buf[2]                       // response
	_ = buf[3]                           // reserved field
	port := int(buf[4])<<8 | int(buf[5]) // proxy listening port

	if resp != BoosterRespSuccess {
		return "", errors.New("booster: remote instance refused hello request")
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", errors.New("booster: " + err.Error())
	}

	paddr := net.JoinHostPort(host, strconv.Itoa(port))

	return paddr, nil
}

func (b *Booster) handleHello(conn net.Conn) error {
	// TODO(daniel): there could be some cases where the hello request should be refused.
	// Atm we always reply ok to this request.
	port := b.Proxy.Port()

	buf := make([]byte, 0, 6)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDHello)
	buf = append(buf, BoosterRespSuccess)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, byte(port>>8), byte(port))

	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write hello response: " + err.Error())
	}

	return nil
}
