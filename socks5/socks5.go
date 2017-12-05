// Package socks5 provides a SOCKS5 server implementation. See RFC 1928
// for protocol specification.
package socks5

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	socks5Version = uint8(5)
)

// Possible METHOD field values
const (
	socks5MethodNoAuth              = uint8(0)
	socks5MethodGSSAPI              = uint8(1)
	socks5MethodUsernamePassword    = uint8(2)
	socks5MethodNoAcceptableMethods = uint8(0xff)
)

// Possible CMD field values
const (
	socks5CmdConnect   = uint8(1)
	socks5CmdBind      = uint8(2)
	socks5CmdAssociate = uint8(3)
)

// Possible REP field values
const (
	socks5RespSuccess                 = uint8(0)
	socks5RespGeneralServerFailure    = uint8(1)
	socks5RespConnectionNotAllowed    = uint8(2)
	socks5RespNetworkUnreachable      = uint8(3)
	socks5RespHostUnreachable         = uint8(4)
	socks5RespConnectionRefused       = uint8(5)
	socks5RespTTLExpired              = uint8(6)
	socks5RespCommandNotSupported     = uint8(7)
	socks5RespAddressTypeNotSupported = uint8(8)
	socks5RespUnassigned              = uint8(9)
)

const (
	// FieldReserved should be used to fill fields marked as reserved.
	socks5FieldReserved = uint8(0x00)
)

const (
	// AddrTypeIPV4 is a version-4 IP address, with a length of 4 octets
	socks5IP4 = uint8(1)

	// AddrTypeFQDN field contains a fully-qualified domain name. The first
	// octet of the address field contains the number of octets of name that
	// follow, there is no terminating NUL octet.
	socks5FQDN = uint8(3)

	// AddrTypeIPV6 is a version-6 IP address, with a length of 16 octets.
	socks5IP6 = uint8(4)
)

var (
	supportedMethods = []uint8{socks5MethodNoAuth}
)

// Dialer is the interface that wraps the DialContext function.
type Dialer interface {
	// DialContext opens a connection to addr, which should
	// be a canonical address with host and port.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// Socks5 represents a SOCKS5 proxy server implementation.
type Socks5 struct {
	*log.Logger

	// Dialer is used when connecting to a remote host. Could
	// be useful when chaining multiple proxies.
	Dialer

	ReadWriteTimeout time.Duration
	ChunkSize        int64

	sync.Mutex
	port              int
	workloadListeners map[string]chan int
	workload          int
}

// NewSOCKS5 returns a new Socks5 instance.
func NewSOCKS5(dialer Dialer, log *log.Logger) *Socks5 {
	s := new(Socks5)
	s.ReadWriteTimeout = 2 * time.Minute
	s.ChunkSize = 4 * 1024
	s.Dialer = dialer
	s.Logger = log

	return s
}

// SOCKS5 returns a new Socks5 instance with default logger and dialer.
func SOCKS5() *Socks5 {
	d := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	log := log.New(os.Stdout, "SOCKS5   ", log.LstdFlags)

	return NewSOCKS5(d, log)
}

// ListenAndServe accepts and handles TCP connections
// using the SOCKS5 protocol.
func (s *Socks5) ListenAndServe(port int) error {
	s.Lock()
	s.port = port
	s.Unlock()

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	s.Printf("listening on port: %v", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Printf("tcp accept error: %v", err)
			continue
		}

		go func() {
			if err := s.Handle(conn); err != nil {
				s.Println(err)
			}
		}()
	}
}

// Handle performs the steps required to be SOCKS5 compliant.
// See RFC 1928 for details.
//
// Should run in its own go routine, closes the connection
// when returning.
func (s *Socks5) Handle(conn net.Conn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer conn.Close()

	// method sub-negotiation phase
	if err := s.Negotiate(conn); err != nil {
		return err
	}

	// request details

	// len is just an estimation
	buf := make([]byte, 6+net.IPv4len)

	if _, err := io.ReadFull(conn, buf[:3]); err != nil {
		return errors.New("socks5: unable to read request: " + err.Error())
	}

	v := buf[0]   // protocol version
	cmd := buf[1] // command to execute
	_ = buf[2]    // reserved field

	// Check version number
	if v != socks5Version {
		return errors.New("socks5: unsupported version: " + string(v))
	}

	target, err := ReadAddress(conn)
	if err != nil {
		return err
	}

	var tconn net.Conn
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch cmd {
	case socks5CmdConnect:
		tconn, err = s.Connect(ctx, conn, target)
	case socks5CmdAssociate:
		tconn, err = s.Associate(ctx, conn, target)
	case socks5CmdBind:
		tconn, err = s.Bind(ctx, conn, target)
	default:
		return errors.New("unexpected CMD(" + strconv.Itoa(int(cmd)) + ")")
	}
	if err != nil {
		return err
	}
	defer tconn.Close()

	// start proxying
	s.pushLoad()
	s.ProxyData(conn, tconn)
	s.popLoad()

	return nil
}

// ProxyData copies data from src to dst and the other way around.
// Closes the connections when they are idle for more than the duration
// described in ReadWriteTimeout.
func (s *Socks5) ProxyData(src net.Conn, dst net.Conn) {
	timer := time.AfterFunc(s.ReadWriteTimeout, func() {
		src.Close()
		dst.Close()
	})

	c := make(chan error, 2)

	// proxy in both directions
	go func(c chan error, src net.Conn, dst net.Conn) {
		for {
			_, err := io.CopyN(dst, src, s.ChunkSize)
			c <- err
		}
	}(c, src, dst)

	go func(c chan error, src net.Conn, dst net.Conn) {
		for {
			_, err := io.CopyN(dst, src, s.ChunkSize)
			c <- err
		}
	}(c, dst, src)

	for err := range c {
		if err != nil {
			// timeout? EOF?
			// TODO(daniel): check if in some cases is better to keep the connection open.
			// It could also be useful to add a connection caching mechanism, something like
			// what http.Transport does.
			return
		}

		// io operations did not return any errors. Reset
		// deadline and keep on transfering data
		timer.Reset(s.ReadWriteTimeout)
	}
}

// ReadAddress reads hostname and port and converts them
// into its string format, properly formatted.
//
// r expects to read one byte that specifies the address
// format (1/3/4), followed by the address itself and a
// 16 bit port number.
//
// addr == "" only when err != nil.
func ReadAddress(r io.Reader) (addr string, err error) {

	// cap is just an estimantion
	buf := make([]byte, 0, 2+net.IPv6len)
	buf = buf[:1]

	if _, err := io.ReadFull(r, buf); err != nil {
		return "", errors.New("unable to read address type: " + err.Error())
	}

	atype := buf[0] // address type

	bytesToRead := 0
	switch atype {
	case socks5IP4:
		bytesToRead = net.IPv4len
	case socks5IP6:
		bytesToRead = net.IPv6len
	case socks5FQDN:
		_, err := io.ReadFull(r, buf[:1])
		if err != nil {
			return "", errors.New("failed to read domain length: " + err.Error())
		}
		bytesToRead = int(buf[0])
	default:
		return "", errors.New("got unknown address type " + strconv.Itoa(int(atype)))
	}

	if cap(buf) < bytesToRead {
		buf = make([]byte, bytesToRead)
	} else {
		buf = buf[:bytesToRead]
	}
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", errors.New("failed to read address: " + err.Error())
	}

	var host string
	if atype == socks5FQDN {
		host = string(buf)
	} else {
		host = net.IP(buf).String()
	}

	if _, err := io.ReadFull(r, buf[:2]); err != nil {
		return "", errors.New("failed to read port: " + err.Error())
	}

	port := int(buf[0])<<8 | int(buf[1])
	addr = net.JoinHostPort(host, strconv.Itoa(port))

	return addr, nil
}
