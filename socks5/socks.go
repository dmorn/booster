package socks5

import (
	"context"
	"io"
)

// Socks5 request/response codes

// Some version numbers
const (
	Version4 = uint8(4)
	Version5 = uint8(5)
)

// Possible METHOD field values
const (
	MethodNoAuth              = uint8(0)
	MethodGSSAPI              = uint8(1)
	MethodUsernamePassword    = uint8(2)
	MethodNoAcceptableMethods = uint8(0xff)
)

// Possible CMD field values
const (
	CmdConnect   = uint8(1)
	CmdBind      = uint8(2)
	CmdAssociate = uint8(3)
)

// Possible REP field values
const (
	RespSucceded                = uint8(0)
	RespGeneralServerFailure    = uint8(1)
	RespConnectionNotAllowed    = uint8(2)
	RespNetworkUnreachable      = uint8(3)
	RespHostUnreachable         = uint8(4)
	RespConnectionRefused       = uint8(5)
	RespTTLExpired              = uint8(6)
	RespCommandNotSupported     = uint8(7)
	RespAddressTypeNotSupported = uint8(8)
	RespUnassigned              = uint8(9)
)

const (
	// FieldReserved should be used to fill fields marked as reserved.
	FieldReserved = uint8(0x00)
)

const (
	// AddrTypeIPV4 is a version-4 IP address, with a length of 4 octets
	AddrTypeIPV4 = uint8(1)

	// AddrTypeDomainName field contains a fully-qualified domain name. The first
	// octet of the address field contains the number of octets of name that
	// follow, there is no terminating NUL octet.
	AddrTypeDomainName = uint8(3)

	// AddrTypeIPV6 is a version-6 IP address, with a length of 16 octets.
	AddrTypeIPV6 = uint8(4)
)

// Socks5er describes a socks5 handler instance.
type Socks5er interface {
	io.ReadWriteCloser

	Runner
	Negotiater
	Connecter
	Associater
	Binder
}

// WriteFunc is just an alias for io.Writer's Write function
type WriteFunc func([]byte) (int, error)

// Negotiater is the interface that wraps the basic Negotiate function.
//
// Negotiate handles socks's method sub-negotiation, checking the requested protocol
// version and also checks if the client asked for a method that is actually supported.
// Writes the response back using w.
type Negotiater interface {
	Negotiate(ctx context.Context, req *NegRequest, w WriteFunc) error
}

// Connecter is the interface that wraps the basic Connect function.
// TODO(daniel): doc
type Connecter interface {
	Connect(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error)
}

// Binder is the interface that wraps the basic Bind function.
// TODO(daniel): doc
type Binder interface {
	Bind(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error)
}

// Associater is the interface that wraps the basic Associate function.
// TODO(daniel): doc
type Associater interface {
	Associate(ctx context.Context, req *Request, w WriteFunc) (io.ReadWriteCloser, error)
}

// Marshaler is the interface that wraps the Marshal function.
//
// Marshal converts it's data into []byte.
type Marshaler interface {
	Marshal() ([]byte, error)
}

// Unmarshaler is the interface that wraps the Unmarshal function.
//
// Unmarshal fills the underlying struct with the data contained in []byte,
// if it is in the right format.
type Unmarshaler interface {
	Unmarshal([]byte) error
}

// Runner is the interface that wraps the Run function.
//
// Run should execute the whole socks5 transaction sequence.
// Ex: Negotiation, connection, proxying.
type Runner interface {
	Run() error
}
