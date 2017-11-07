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

		go func() {
			s5 := socks5.NewSocks5(conn)
			if err := s5.Run(); err != nil {
				s.Log.Println(err)
			}
		}()
	}
}
