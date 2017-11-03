package socks5

import (
	"context"
	"fmt"
	"io"
)

type Socks5 struct {
	Socks5er

	rwc              io.ReadWriteCloser // such as conn.Conn
	trg              io.ReadWriteCloser // such as conn.Conn
	supportedMethods []uint8
}

var _ Socks5er = &Socks5{}

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

func (s *Socks5) Run() error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	defer s.rwc.Close()

	// First negotiate
	buf := make([]byte, 257)
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

	// execute proper command
	var err error
	var trg io.ReadWriteCloser // such as conn.Conn

	switch req.Cmd {
	case CmdConnect:
		trg, err = s.Connect(ctx, req, s.Write)
	case CmdAssociate:
		trg, err = s.Associate(ctx, req, s.Write)
	case CmdBind:
		trg, err = s.Associate(ctx, req, s.Write)
	default:
		return fmt.Errorf("unexpected CMD(%v)", req.Cmd)
	}
	if err != nil {
		return err
	}

	defer func(rwc io.ReadWriteCloser) {
		if conn, ok := rwc.(io.Closer); ok {
			conn.Close()
		}
	}(trg)

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

	res := new(Response)
	res.Ver = req.Ver
	res.Rep = RespCommandNotSupported
	res.Rsv = FieldReserved
	res.AddrType = req.AddrType
	res.BndAddr = req.Addr
	res.BndPort = req.DstPort

	mr, err := res.Marshal()
	if err != nil {
		return nil, err
	}

	c := make(chan error, 1)
	go func(c chan<- error, p []byte) {
		_, err := w(p)
		c <- err
	}(c, mr)

	select {
	case <-ctx.Done():
		<-c
		return nil, ctx.Err()
	case err := <-c:
		return nil, err
	}
}

func (s *Socks5) Bind(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("unsupported method")
}

func (s *Socks5) Associate(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("unsupported method")
}
