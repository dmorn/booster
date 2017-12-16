package node

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/danielmorandini/booster-network/socks5"
)

// RemoteNode represents a remote booster node.
type RemoteNode struct {
	ID     string // sha1 string representation
	Host   string
	Pport  string // Proxy port
	Bport  string // Booster port
	cancel context.CancelFunc

	sync.Mutex
	IsActive bool // set to false when connection is nil
	workload int
}

// NewRemoteNode create a new RemoteNode instance.
func NewRemoteNode(host, pport, bport string) *RemoteNode {
	n := new(RemoteNode)
	n.Host = host
	n.Pport = pport
	n.Bport = bport

	// id is the sha1 of host + bport + pport
	h := sha1.New()
	h.Write([]byte(host))
	h.Write([]byte(bport))
	h.Write([]byte(pport))
	n.ID = fmt.Sprintf("%x", h.Sum(nil))

	return n
}

func (n *RemoteNode) String() string {
	baddr := net.JoinHostPort(n.Host, n.Bport)
	paddr := net.JoinHostPort(n.Host, n.Pport)
	return fmt.Sprintf("node (%v): booster @ %v, proxy @ %v, active: %v", n.ID, baddr, paddr, n.IsActive)
}

func ReadRemoteNode(r io.Reader) (*RemoteNode, error) {
	buf := make([]byte, 20) // sha1 len
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("remote node: " + err.Error())
	}

	id := fmt.Sprintf("%x", buf)

	host, err := socks5.ReadHost(r)
	if err != nil {
		return nil, errors.New("remote node: " + err.Error())
	}
	pport, err := socks5.ReadPort(r)
	if err != nil {
		return nil, errors.New("remote node: " + err.Error())
	}
	bport, err := socks5.ReadPort(r)
	if err != nil {
		return nil, errors.New("remote node: " + err.Error())
	}

	buf = buf[:2]
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("remote node: " + err.Error())
	}

	isActive := buf[0]
	workload := buf[1]

	return &RemoteNode{
		ID:       id,
		Host:     host,
		Pport:    pport,
		Bport:    bport,
		IsActive: int(isActive) != 0,
		workload: int(workload),
	}, nil
}

func (n *RemoteNode) EncodeBinary() ([]byte, error) {
	idbuf, err := hex.DecodeString(n.ID)
	hbuf, err := socks5.EncodeHostBinary(n.Host)   // host buffer
	ppbuf, err := socks5.EncodePortBinary(n.Pport) // proxy port buffer
	bpbuf, err := socks5.EncodePortBinary(n.Bport) // booster port buffer
	if err != nil {
		return nil, errors.New("remote node: " + err.Error())
	}

	n.Lock()
	load := n.workload
	n.Unlock()

	if load > 0xff {
		return nil, errors.New("remote node: load out of range: " + strconv.Itoa(load))
	}

	buf := make([]byte, 0, len(idbuf)+len(hbuf)+len(ppbuf)+len(bpbuf))
	buf = append(buf, idbuf...)
	buf = append(buf, hbuf...)
	buf = append(buf, ppbuf...)
	buf = append(buf, bpbuf...)
	buf = strconv.AppendBool(buf, n.IsActive)
	buf = append(buf, byte(load))

	return buf, nil
}

// StartUpdating expects conn to produce booster status messages. It then
// uses that data to update the workload's value of the node.
// It also adds a cancel function to the node, that can be used to make
// the updating stop.
//
// If the connection is closed, the data is somehow corrupted or a cancel
// signal is received, it closes the connection and sets the IsActive value
// of the node to false.
func (n *RemoteNode) StartUpdating(conn net.Conn) error {
	if conn == nil {
		return errors.New("remote node: found nil connection. Unable to update node status")
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.IsActive = true
	n.cancel = cancel

	go func() {
		buf := make([]byte, 3)
		c := make(chan error)

		go func() {
			for {
				if _, err := io.ReadFull(conn, buf); err != nil {
					c <- err
					return
				}

				_ = buf[0]     // version - already checked in the hello procedure
				_ = buf[1]     // reserved field
				load := buf[2] // workload

				n.Lock()
				n.workload = int(load)
				n.Unlock()
			}
		}()

		fail := func() {
			conn.Close()
			n.IsActive = false
			n.cancel = nil
		}

		// in any case we're done
		select {
		case <-ctx.Done():
			fail()
			return
		case <-c:
			fail()
			return
		}
	}()

	return nil
}
