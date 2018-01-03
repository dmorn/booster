package node

import (
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

// Operations on RemoteNode
const (
	OpUpdated = iota
	OpRemoved
	OpCreated
)

// RemoteNode represents a remote booster node.
type RemoteNode struct {
	ID    string // sha1 string representation
	Host  string
	Pport string // Proxy port
	Bport string // Booster port

	sync.Mutex
	IsActive      bool // set to false when connection is nil
	workload      int
	lastOperation int // last operation made on this node
}

// NewRemoteNode create a new RemoteNode instance.
func NewRemoteNode(host, pport, bport string) *RemoteNode {
	n := new(RemoteNode)
	n.Host = host
	n.Pport = pport
	n.Bport = bport
	n.workload = 0

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
	n.Lock()
	wl := n.workload
	n.Unlock()

	return fmt.Sprintf("node (%v): booster @ %v, proxy @ %v, workload: %v, active: %v, lastOp: %v", n.ID, baddr, paddr, wl, n.IsActive, n.lastOperation)
}

// ReadRemoteNode reads from reader expecting it to contain a remote node.
func ReadRemoteNode(r io.Reader) (*RemoteNode, error) {
	buf := make([]byte, 20) // sha1 len
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("remote node: unable to read identifier: " + err.Error() + " buffer: " + fmt.Sprintf("%v", buf))
	}

	id := fmt.Sprintf("%x", buf)
	host, err := socks5.ReadHost(r)
	if err != nil {
		return nil, errors.New("remote node: unable to decode host: " + err.Error())
	}
	pport, err := socks5.ReadPort(r)
	if err != nil {
		return nil, errors.New("remote node: unable to decode p port: " + err.Error())
	}
	bport, err := socks5.ReadPort(r)
	if err != nil {
		return nil, errors.New("remote node: unable to decode b port: " + err.Error())
	}

	buf = buf[:3]
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("remote node: unable to decode state: " + err.Error())
	}

	isActive := buf[0]
	workload := int(buf[1])
	lastOp := int(buf[2])

	return &RemoteNode{
		ID:            id,
		Host:          host,
		Pport:         pport,
		Bport:         bport,
		IsActive:      int(isActive) != 0,
		workload:      workload,
		lastOperation: lastOp,
	}, nil
}

// EncodeBinary encodes the remote node into its binary
// representation.
func (n *RemoteNode) EncodeBinary() ([]byte, error) {
	if n == nil {
		return nil, errors.New("remote node: trying to encode nil")
	}

	idbuf, err := hex.DecodeString(n.ID)
	hbuf, err := socks5.EncodeHostBinary(n.Host)   // host buffer
	ppbuf, err := socks5.EncodePortBinary(n.Pport) // proxy port buffer
	bpbuf, err := socks5.EncodePortBinary(n.Bport) // booster port buffer
	if err != nil {
		return nil, errors.New("remote node: unable to encode: " + err.Error())
	}

	n.Lock()
	load := n.workload
	lastOp := n.lastOperation
	n.Unlock()
	if load > 0xff {
		return nil, errors.New("remote node: load out of range: " + strconv.Itoa(load))
	}

	isActive := 0
	if n.IsActive {
		isActive = 1
	}

	buf := make([]byte, 0, len(idbuf)+len(hbuf)+len(ppbuf)+len(bpbuf)+3)
	buf = append(buf, idbuf...)
	buf = append(buf, hbuf...)
	buf = append(buf, ppbuf...)
	buf = append(buf, bpbuf...)
	buf = append(buf, byte(isActive))
	buf = append(buf, byte(load))
	buf = append(buf, byte(lastOp))

	return buf, nil
}
