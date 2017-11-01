package socks5

import (
	"context"
	"fmt"
)

type negRequest struct {
	Unmarshaler

	Ver     uint8
	Methods []uint8
}

// Unmarshal fill r with data contained in p.
// expected input format:
//
// +----+----------+----------+
// |VER | NMETHODS | METHODS  |
// +----+----------+----------+
// | 1  |    1     | 1 to 255 |
// +----+----------+----------+
// numers represent field size
//
func (r *negRequest) Unmarshal(p []byte) error {
	v := p[0] // version

	// procede with method checking
	nmethods := uint8(p[1])        // methods field size
	methods := p[2:(2 + nmethods)] // methods supported by client

	r.Ver = v
	r.Methods = methods
	return nil
}

type negResponse struct {
	Marshaler

	Ver    uint8
	Method uint8
}

// Marshal converts r into []byte.
// Output format:
//
// +----+--------+
// |VER | METHOD |
// +----+--------+
// | 1  |   1    |
// +----+--------+
// numers represent field size
//
func (r *negResponse) Marshal() ([]byte, error) {
	return []byte{r.Ver, r.Method}, nil
}

var _ Negotiater = &Socks5{}

// Negotiate expects s.Read to fill a buffer with a well formed negRequest.
// Returns an error if versions is different from 5. Writes results back using w
// function.
func (s *Socks5) Negotiate(ctx context.Context, w WriteFunc) error {
	req := &negRequest{}
	buf := make([]byte, 257)
	c := make(chan error, 1)

	if _, err := s.Read(buf); err != nil {
		return err
	}

	if err := req.Unmarshal(buf); err != nil {
		return err
	}

	// Check version number
	if req.Ver != Version5 {
		return fmt.Errorf("unsupported version (%v)", req.Ver)
	}

	// build response checking methods
	resp := &negResponse{
		Ver:    req.Ver,
		Method: s.methodsSupported(req.Methods),
	}

	mr, err := resp.Marshal()
	if err != nil {
		return err
	}

	go func(c chan<- error) {
		_, err := w(mr)
		c <- err
	}(c)

	select {
	case <-ctx.Done():
		<-c // wait w to return
		return ctx.Err()
	case err := <-c:
		return err
	}
}

func (s *Socks5) methodsSupported(m []uint8) uint8 {
	for _, sm := range s.supportedMethods {
		for _, tm := range m {
			if sm == tm {
				return sm
			}
		}
	}

	return MethodNoAcceptableMethods
}
