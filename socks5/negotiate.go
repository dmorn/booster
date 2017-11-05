package socks5

import (
	"context"
	"fmt"
)

type NegRequest struct {
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
// numbers represent field size
//
func (r *NegRequest) Unmarshal(p []byte) error {
	v := p[0] // version

	// procede with method checking
	nmethods := uint8(p[1])        // methods field size
	methods := p[2:(2 + nmethods)] // methods supported by client

	r.Ver = v
	r.Methods = methods
	return nil
}

type NegResponse struct {
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
func (r *NegResponse) Marshal() ([]byte, error) {
	return []byte{r.Ver, r.Method}, nil
}

var _ Negotiater = &Socks5{}

// Negotiate expects s.Read to fill a buffer with a well formed negRequest.
// Returns an error if versions is different from 5. Writes results back using w
// function.
func (s *Socks5) Negotiate(ctx context.Context, req *NegRequest, w WriteFunc) error {
	// Check version number
	if req.Ver != Version5 {
		return fmt.Errorf("unsupported version (%v)", req.Ver)
	}

	// build response checking methods
	resp := &NegResponse{
		Ver:    req.Ver,
		Method: s.areMethodsSupported(req.Methods),
	}

	mr, err := resp.Marshal()
	if err != nil {
		return err
	}

	c := make(chan error, 1)
	go func(c chan<- error, p []byte) {
		_, err := w(p)
		c <- err
	}(c, mr)

	select {
	case <-ctx.Done():
		<-c // wait w to return
		return ctx.Err()
	case err := <-c:
		return err
	}
}

func (s *Socks5) areMethodsSupported(m []uint8) uint8 {
	for _, sm := range s.supportedMethods {
		for _, tm := range m {
			if sm == tm {
				return sm
			}
		}
	}

	return MethodNoAcceptableMethods
}
