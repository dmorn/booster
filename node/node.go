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
	"time"

	"github.com/danielmorandini/booster-network/socks5"
)

// Node represents a remote booster node.
type Node struct {
	id    string // sha1 string representation
	BAddr net.Addr
	PAddr net.Addr

	sync.Mutex
	cancel   context.CancelFunc // added when some goroutin is updating its workload.
	IsActive bool               // set to false when connection is nil
	workload int

	lastOperation *operation // last operation made on this node
}

type operation struct {
	id string // sha1 identifier
	op uint8
}

func (o *operation) String() string {
	switch o.op {
	case BoosterNodeAdded:
		return "added"
	case BoosterNodeClosed:
		return "closed"
	case BoosterNodeRemoved:
		return "removed"
	case BoosterNodeUpdated:
		return fmt.Sprintf("updated (%v)", o.id)
	default:
		return "unknown"
	}
}

// NewNode create a new Node instance.
func NewNode(host, pport, bport string) (*Node, error) {
	n := new(Node)
	baddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, bport))
	if err != nil {
		return nil, errors.New("node: unable to create baddr: " + err.Error())
	}
	n.BAddr = baddr

	paddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, pport))
	if err != nil {
		return nil, errors.New("node: unable to create paddr: " + err.Error())
	}
	n.PAddr = paddr

	n.workload = 0
	n.lastOperation = new(operation)

	// id is the sha1 of host + bport + pport
	n.id = sha1Hash([]byte(host), []byte(bport), []byte(pport))

	return n, nil
}

// Desc returns the description of the node in a multiline string.
func (n *Node) String() string {
	n.Lock()
	wl := n.workload
	op := n.lastOperation
	n.Unlock()

	activeStr := "inactive"
	if n.IsActive {
		activeStr = "active"
	}

	host, bport, _ := net.SplitHostPort(n.BAddr.String())
	_, pport, _ := net.SplitHostPort(n.PAddr.String())

	return fmt.Sprintf("[node (%v), @%v(b%v-p%v), %v]: wl: %v, lastop: %v", n.ID(), host, bport, pport, activeStr, wl, op.String())
}

// ID returns the id of the node. Required by tracer.Pinger in this case.
func (n *Node) ID() string {
	return n.id
}

// Close calls the cancel function if present, then sets active state to false.
func (n *Node) Close() error {
	n.Lock()
	defer n.Unlock()
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
	n.IsActive = false
	n.lastOperation.op = BoosterNodeClosed

	return nil
}

// LastOperation returns the last operation code of the node.
func (n *Node) LastOperation() uint8 {
	return n.lastOperation.op
}

// ReadNode reads from reader expecting it to contain a node.
func ReadNode(r io.Reader) (*Node, error) {
	buf := make([]byte, 20) // sha1 len
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("node: unable to read identifier: " + err.Error() + " buffer: " + fmt.Sprintf("%v", buf))
	}

	id := fmt.Sprintf("%x", buf)
	host, err := socks5.ReadHost(r)
	if err != nil {
		return nil, errors.New("node: unable to decode host: " + err.Error())
	}
	pport, err := socks5.ReadPort(r)
	if err != nil {
		return nil, errors.New("node: unable to decode p port: " + err.Error())
	}
	bport, err := socks5.ReadPort(r)
	if err != nil {
		return nil, errors.New("node: unable to decode b port: " + err.Error())
	}

	buf = buf[:3]
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("node: unable to decode state: " + err.Error())
	}

	isActive := buf[0]
	workload := int(buf[1])
	lastOp := buf[2]

	buf = buf[:20]
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, errors.New("node: unable to decode last operation id: " + err.Error())
	}
	lastOpID := fmt.Sprintf("%x", buf)

	node, err := NewNode(host, pport, bport)
	if err != nil {
		return nil, err
	}

	node.id = id
	node.IsActive = int(isActive) != 0
	node.workload = workload
	node.lastOperation = &operation{
		id: lastOpID,
		op: lastOp,
	}

	return node, nil
}

// EncodeBinary encodes the node into its binary
// representation.
func (n *Node) EncodeBinary() ([]byte, error) {
	if n == nil {
		return nil, errors.New("node: trying to encode nil")
	}

	host, bport, err := net.SplitHostPort(n.BAddr.String())
	_, pport, err := net.SplitHostPort(n.PAddr.String())
	if err != nil {
		return nil, errors.New("node: unable to split address: " + err.Error())
	}

	idbuf, err := hex.DecodeString(n.ID())
	hbuf, err := socks5.EncodeHostBinary(host)   // host buffer
	ppbuf, err := socks5.EncodePortBinary(pport) // proxy port buffer
	bpbuf, err := socks5.EncodePortBinary(bport) // booster port buffer
	if err != nil {
		return nil, errors.New("node: unable to encode: " + err.Error())
	}

	n.Lock()
	load := n.workload
	lastOp := n.lastOperation.op
	opidbuf, err := hex.DecodeString(n.lastOperation.id)
	n.Unlock()

	if err != nil {
		opidbuf = make([]byte, 20) // just put a fake hash
	}
	// It could happen that we do not have any operation id
	if len(opidbuf) != 20 {
		opidbuf = make([]byte, 20) // just put a fake hash
	}

	if load > 0xff {
		return nil, errors.New("node: load out of range: " + strconv.Itoa(load))
	}

	isActive := 0
	if n.IsActive {
		isActive = 1
	}

	buf := make([]byte, 0, len(idbuf)+len(hbuf)+len(ppbuf)+len(bpbuf)+3+len(opidbuf))
	buf = append(buf, idbuf...)
	buf = append(buf, hbuf...)
	buf = append(buf, ppbuf...)
	buf = append(buf, bpbuf...)
	buf = append(buf, byte(isActive))
	buf = append(buf, byte(load))
	buf = append(buf, byte(lastOp))
	buf = append(buf, opidbuf...)

	return buf, nil
}

// Ping dials with the node with little timeout. Returns an error
// if the endpoint is not reachable, nil otherwise. Required by tracer.Pinger.
func (n *Node) Ping(ctx context.Context) error {
	if n.IsActive {
		return errors.New("connection already enstablished")
	}

	d := net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 0 * time.Second,
	}
	_, err := d.DialContext(ctx, n.BAddr.Network(), n.BAddr.String())

	return err
}

func (n *Node) Addr() net.Addr {
	return n.BAddr
}

func sha1Hash(images ...[]byte) string {
	h := sha1.New()
	for _, image := range images {
		h.Write(image)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

