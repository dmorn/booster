package proxy

import (
	"log"
	"net"
	"strconv"

	"github.com/danielmorandini/booster/socks5"
)

// Server acts as forward proxy wrapper
type Server struct {
	Port int

	Log *log.Logger
}

// ListenAndServe accepts tcp connections and forwards them
// using the underling proxy imlementation
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Log.Printf("[TCP Accept Error]: %v\n", err)
			continue
		}
		s5 := socks5.NewSocks5(conn)
		go s.handle(s5)
	}
}

func (s *Server) handle(proxy socks5.Socks5er) {
	if err := proxy.Run(); err != nil {
		s.Log.Println(err)
	}
}
