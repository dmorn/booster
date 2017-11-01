package socks5

import (
	"bufio"
	"context"
	"io"
)

type Socks5 struct {
	Socks5er

	rwc              io.ReadWriteCloser // such as conn.Conn
	supportedMethods []uint8
}

func NewSocks5(rwc io.ReadWriteCloser) *Socks5 {
	return &Socks5{
		rwc:              rwc,
		supportedMethods: []uint8{MethodNoAuth},
	}
}

func (s *Socks5) Close() error {
	return s.rwc.Close()
}

func (s *Socks5) Run() error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// First negotiate
	if err := s.Negotiate(ctx, s.Write); err != nil {
		return err
	}

	return nil
}

func (s *Socks5) Write(p []byte) (int, error) {
	w := bufio.NewWriter(s.rwc)
	return w.Write(p)
}

func (s *Socks5) Read(p []byte) (int, error) {
	r := bufio.NewReader(s.rwc)
	return r.Read(p)
}
