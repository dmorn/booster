/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package socks5 provides a SOCKS5 server implementation. See RFC1928
// for protocol specification.
package socks5

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/pubsub"
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

const (
	// TopicTunnelEvents is the topic where the tunnels updates are published
	TopicTunnelEvents = "topic_tunnel_events"

	// TopicNet is the topic where bandwidth usage updates are published
	TopicNet = "topic_network"
)

type Event int

// Possible proxy operations.
const (
	EventPush Event = iota
	EventPop
)

// PubSub describes the required functionalities of a publication/subscription object.
type PubSub interface {
	Sub(cmd *pubsub.Command) (pubsub.CancelFunc, error)
	Pub(message interface{}, topic string) error
}

// Dialer is the interface that wraps the DialContext function.
type Dialer interface {
	// DialContext opens a connection to addr, which should
	// be a canonical address with host and port.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// Socks5 represents a SOCKS5 proxy server implementation.
type Socks5 struct {
	PubSub

	port int

	ReadWriteTimeout time.Duration
	ChunkSize        int64
	BandwidthCheck   time.Duration
	NextCopyDelay    time.Duration
	DownloadIO       *network.NetworkIO
	UploadIO         *network.NetworkIO

	stop chan struct{}

	sync.Mutex
	Dialer
}

type BandwidthMessage struct {
	Bandwidth int
	Tot       int
	D         time.Duration
	Download  bool
}

// SOCKS5 returns a new Socks5 instance with default logger and dialer.
func New(d Dialer) *Socks5 {
	s := new(Socks5)
	ps := pubsub.New()
	bc := time.Second
	ncd := time.Second * 0

	dIO := new(network.NetworkIO)
	uIO := new(network.NetworkIO)
	f := func(bIO *network.NetworkIO, dl bool) func() {
		return func() {
			bIO.Lock()
			b := bIO.Bandwidth
			n := bIO.N
			bIO.Unlock()

			s.Pub(&BandwidthMessage{
				Bandwidth: int(b),
				Tot:       int(n),
				D:         bc,
				Download:  dl,
			}, TopicNet)
		}
	}

	dIO.TickerFunc(bc, f(dIO, true))
	dIO.NextCopyDelay = ncd
	uIO.TickerFunc(bc, f(uIO, false))
	uIO.NextCopyDelay = ncd

	s.ReadWriteTimeout = 2 * time.Minute
	s.BandwidthCheck = bc
	s.NextCopyDelay = ncd
	s.ChunkSize = 1 * 1024
	s.Dialer = d
	s.PubSub = ps
	s.stop = make(chan struct{})
	s.DownloadIO = dIO
	s.UploadIO = uIO

	return s
}

func (s *Socks5) Run(port int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := make(chan error)
	go func() {
		errc <- s.ListenAndServe(ctx, port)
	}()

	select {
	case err := <-errc:
		return err
	case <-s.stop:
		cancel()
		s.DownloadIO.Ticker.Stop()
		s.UploadIO.Ticker.Stop()

		<-errc // wait for ListenAndServe to return
		return fmt.Errorf("socks5: stopped")
	}
}

func (s *Socks5) Close() error {
	s.stop <- struct{}{}

	return nil
}

// ListenAndServe accepts and handles TCP connections
// using the SOCKS5 protocol.
func (s *Socks5) ListenAndServe(ctx context.Context, port int) error {
	p := strconv.Itoa(port)
	ln, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Info.Printf("socks5: listening on port: %v", port)

	errc := make(chan error)
	defer close(errc)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errc <- fmt.Errorf("socks5: cannot accept conn: %v", err)
				return
			}

			go s.Handle(ctx, conn)
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		ln.Close()
		<-errc // wait for listener to return
		return ctx.Err()
	}
}

// Handle performs the steps required to be SOCKS5 compliant.
// See RFC 1928 for details.
//
// Should run in its own go routine, closes the connection
// when returning.
func (s *Socks5) Handle(ctx context.Context, conn net.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
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
		return errors.New("socks5: unexpected CMD(" + strconv.Itoa(int(cmd)) + ")")
	}
	if err != nil {
		return err
	}
	defer tconn.Close()

	// start proxying
	s.pubUpdate(tconn.RemoteAddr().String(), protocol.TunnelAdd)
	s.ProxyData(conn, tconn)
	s.pubUpdate(tconn.RemoteAddr().String(), protocol.TunnelRemove)

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
	go func() {
		for {
			// download
			_, err := s.DownloadIO.CopyN(src, dst, s.ChunkSize)
			c <- err
		}
	}()

	go func() {
		for {
			// upload
			_, err := s.UploadIO.CopyN(dst, src, s.ChunkSize)
			c <- err
		}
	}()

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
	host, err := ReadHost(r)
	if err != nil {
		return "", err
	}

	port, err := ReadPort(r)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, port), nil
}

// ReadHost deals with the host part of ReadAddress.
func ReadHost(r io.Reader) (string, error) {
	// cap is just an estimantion
	buf := make([]byte, 0, 1+net.IPv6len)
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

	return host, nil
}

// ReadPort deals with the port part of ReadAddress.
func ReadPort(r io.Reader) (string, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", errors.New("failed to read port: " + err.Error())
	}

	port := int(buf[0])<<8 | int(buf[1])
	return strconv.Itoa(port), nil
}

// EncodeAddressBinary expects as input a canonical host:port address and
// returns the binary representation as speccified in the socks5 protocol (RFC1928).
// Booster uses the same encoding.
func EncodeAddressBinary(addr string) ([]byte, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("booster: unrecognised address format : " + addr + " : " + err.Error())
	}

	hbuf, err := EncodeHostBinary(host)
	if err != nil {
		return nil, err
	}

	pbuf, err := EncodePortBinary(port)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 0, len(hbuf)+len(pbuf))
	buf = append(buf, hbuf...)
	buf = append(buf, pbuf...)

	return buf, nil
}

// EncodeHostBinary encodes a canonical host (IPv4, IPv6, FQDN) into a
// byte slice. Format follows RFC1928.
func EncodeHostBinary(host string) ([]byte, error) {
	buf := make([]byte, 0, 1+len(host)) // 1 if fqdn (address size)

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, socks5IP4)
			ip = ip4
		} else {
			buf = append(buf, socks5IP6)
		}
		buf = append(buf, ip...)
	} else {
		if len(host) > 255 {
			return nil, errors.New("socks5: destination host name too long: " + host)
		}
		buf = append(buf, socks5FQDN)
		buf = append(buf, byte(len(host)))
		buf = append(buf, host...)
	}

	return buf, nil
}

// EncodePortBinary encodes a canonical port into 2 bytes.
// Format follows RFC1928.
func EncodePortBinary(port string) ([]byte, error) {
	buf := make([]byte, 0, 2)
	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New("socks5: failed to parse port number: " + port)
	}
	if p < 1 || p > 0xffff {
		return nil, errors.New("socks5: port number out of range: " + port)
	}

	buf = append(buf, byte(p>>8), byte(p))
	return buf, nil
}

// Port safely returns proxy's listening port.
func (s *Socks5) Port() int {
	s.Lock()
	defer s.Unlock()

	return s.port
}

// Proto returns the protocol used by the proxy in string format.
func (s *Socks5) Proto() string {
	return "socks5"
}

func (s *Socks5) pubUpdate(target string, op protocol.Operation) error {
	p := protocol.PayloadProxyUpdate{
		Target:    target,
		Operation: op,
	}
	err := s.Pub(p, TopicTunnelEvents)
	if err != nil {
		log.Error.Printf("socks5: unable to publish message: %v", err)
	}

	return err
}

func sha1Hash(images ...[]byte) string {
	h := sha1.New()
	for _, image := range images {
		h.Write(image)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
