package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
)

// Connect performs the steps required to pair with a remote node.
// laddr is the local booster address to dial with. raddr is the remote
// node address that as to be registered.
//
// Returns the id of the connected node.
func (b *Booster) Connect(ctx context.Context, network, laddr, raddr string) (string, error) {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := b.DialContext(_ctx, network, laddr)
	if err != nil {
		return "", errors.New("booster: unable to contact booster " + laddr + " : " + err.Error())
	}

	abuf, err := EncodeAddressBinary(raddr)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 0, 3+len(abuf))
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDConnect)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, abuf...)

	if _, err := conn.Write(buf); err != nil {
		return "", errors.New("booster: unable to write connect request: " + err.Error())
	}

	buf = make([]byte, 4+20) // sha1 len
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", errors.New("booster: unable to read connect response: " + err.Error())
	}

	v := buf[0] // version
	if v != BoosterVersion1 {
		return "", errors.New("booster: unsupported booster version in connect response: " + string(v))
	}

	_ = buf[1]  // cmd
	r := buf[2] // response
	if r != BoosterRespAccepted {
		return "", errors.New("booster: connect request refused")
	}

	_ = buf[3] // reserved field
	id := fmt.Sprintf("%x", buf[4:])

	return id, nil
}

func (b *Booster) handleConnect(ctx context.Context, conn net.Conn) error {
	addr, err := socks5.ReadAddress(conn)
	if err != nil {
		return errors.New("booster: " + err.Error())
	}

	bconn, paddr, err := b.Hello(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	host, bport, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("booster: unable to handle connect: " + err.Error())
	}
	_, pport, err := net.SplitHostPort(paddr)
	if err != nil {
		return errors.New("booster: unable to handle connect: " + err.Error())
	}

	id, err := b.AddNode(host, pport, bport, bconn) // node sha1 representation
	if err != nil {
		return err
	}

	bid, err := hex.DecodeString(id)
	if err != nil {
		return errors.New("booster: " + err.Error())
	}

	buf := make([]byte, 0, len(bid)+4)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDConnect)
	buf = append(buf, BoosterRespAccepted)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, bid...)

	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write connect response: " + err.Error())
	}

	return nil
}

// EncodeAddressBinary expects as input a canonical host:port address and
// returns the binary representation as speccified in the socks5 protocol (RFC1928).
// Booster uses the same encoding.
func EncodeAddressBinary(addr string) ([]byte, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("booster: unrecognised address format : " + addr + " : " + err.Error())
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, errors.New("booster: failed to parse port number: " + portStr)
	}
	if port < 1 || port > 0xffff {
		return nil, errors.New("booster: port number out of range: " + portStr)
	}

	buf := make([]byte, 0, 3+len(host)) // 2 for the port, 1 if fqdn (address size)

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, BoosterAddrIP4)
			ip = ip4
		} else {
			buf = append(buf, BoosterAddrIP6)
		}
		buf = append(buf, ip...)
	} else {
		if len(host) > 255 {
			return nil, errors.New("booster: destination host name too long: " + host)
		}
		buf = append(buf, BoosterAddrFQDN)
		buf = append(buf, byte(len(host)))
		buf = append(buf, host...)
	}
	buf = append(buf, byte(port>>8), byte(port))

	return buf, nil
}
