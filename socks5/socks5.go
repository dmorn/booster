package socks5

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
)

// Socks5 is a Socks5er implementation.
type Socks5 struct {
	rwc              io.ReadWriteCloser // such as conn.Conn
	supportedMethods []uint8
}

// NewSocks5 creates a new instance of Socks5.
//
// rwc will be used as source.
func NewSocks5(rwc io.ReadWriteCloser) *Socks5 {
	return &Socks5{
		rwc:              rwc,
		supportedMethods: []uint8{MethodNoAuth},
	}
}

func (s *Socks5) Write(p []byte) (int, error) {
	return s.rwc.Write(p)
}

func (s *Socks5) Read(p []byte) (int, error) {
	return s.rwc.Read(p)
}

// Run marshals and unmarshals requests, starting from the
// negotiation. It then checks which Command the client asked to
// execute and performs the request.
//
// When the negotation phase ends, enstablishes a connection with the remote host,
// then starts proxing requests between client and server.
func (s *Socks5) Run() error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	defer s.rwc.Close()

	// First negotiate
	buf := make([]byte, 256)
	negReq := new(NegRequest)
	if _, err := s.Read(buf); err != nil {
		return err
	}

	if err := negReq.Unmarshal(buf); err != nil {
		return err
	}

	if err := s.Negotiate(ctx, negReq, s.Write); err != nil {
		return err
	}

	// parse request
	p := make([]byte, 256)
	if _, err := s.Read(p); err != nil {
		return err
	}

	req := new(Request)
	if err := req.Unmarshal(p); err != nil {
		return err
	}

	// resolve destination address if it is a FQDN
	dest := req.DestAddr
	if dest.FQDN != "" {
		ipaddr, err := net.ResolveIPAddr("ip", dest.FQDN)
		if err != nil {
			resp, err := NewResponse(nil, RespHostUnreachable)
			if err != nil {
				return fmt.Errorf("unable to build resp message")
			}
			s.writeResp(ctx, resp, s.Write)
			return fmt.Errorf("failed to resolve destination '%v': %v", dest.FQDN, err)
		}
		dest.IP = ipaddr.IP
	}

	// Apply any address rewrites
	req.DestAddr = dest

	// execute proper command
	var err error
	var trg io.ReadWriteCloser // such as conn.Conn

	switch req.Cmd {
	case CmdConnect:
		trg, err = s.Connect(ctx, req, s.Write)
	case CmdAssociate:
		trg, err = s.Associate(ctx, req, s.Write)
	case CmdBind:
		trg, err = s.Bind(ctx, req, s.Write)
	default:
		return fmt.Errorf("unexpected CMD(%v)", req.Cmd)
	}
	if err != nil {
		return err
	}
	defer trg.Close()

	// check that the connection is not nil
	if trg == nil {
		return fmt.Errorf("unable to enstablish connection with remote host")
	}

	// start proxying
	go io.Copy(trg, s.rwc)
	io.Copy(s.rwc, trg)

	return nil
}

func (s *Socks5) Connect(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error) {
	fmt.Printf("connect request %v\n", req)

	target, err := s.DialContext(ctx, "tcp", req.DestAddr.Address())
	if err != nil {
		msg := err.Error()
		rc := RespHostUnreachable
		if strings.Contains(msg, "refused") {
			rc = RespConnectionRefused
		} else if strings.Contains(msg, "network is unreachable") {
			rc = RespNetworkUnreachable
		}

		resp, err := NewResponse(nil, rc)
		if err != nil {
			return nil, fmt.Errorf("unable to build resp message")
		}

		return nil, s.writeResp(ctx, resp, w)
	}

	// Send success
	local := target.LocalAddr().(*net.TCPAddr)
	bind := Addr{IP: local.IP, Port: local.Port}
	resp, err := NewResponse(&bind, RespSucceded)
	if err != nil {
		return nil, err
	}

	if err := s.writeResp(ctx, resp, w); err != nil {
		return nil, err
	}

	return target, nil
}

func (s *Socks5) Bind(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error) {
	resp, err := NewResponse(nil, RespCommandNotSupported)
	if err != nil {
		return nil, err
	}

	if err = s.writeResp(ctx, resp, w); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("unsupported method")
}

func (s *Socks5) Associate(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error) {
	resp, err := NewResponse(nil, RespCommandNotSupported)
	if err != nil {
		return nil, err
	}

	if err = s.writeResp(ctx, resp, w); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("unsupported method")
}

func (s *Socks5) Dial(network, addr string) (c net.Conn, err error) {
	return net.Dial(network, addr)
}

func (s *Socks5) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	c := make(chan net.Conn, 1)
	errc := make(chan error, 1)

	go func(c chan net.Conn, errc chan error) {
		conn, err := s.Dial(network, addr)
		if err != nil {
			errc <- err
			return
		}

		c <- conn
	}(c, errc)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-c:
		return conn, nil
	case err := <-errc:
		return nil, err
	}
}

func (s *Socks5) writeResp(ctx context.Context, resp *Response, w WriteFunc) error {
	c := make(chan error, 1)
	go func(c chan error) {
		b, err := resp.Marshal()
		if err != nil {
			c <- err
			return
		}

		_, err = w(b)
		c <- err
	}(c)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}
