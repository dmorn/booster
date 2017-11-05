package socks5

import (
	"fmt"
	"net"
	"strconv"
)

// Addr is used to return the target Addr
// which may be specified as IPv4, IPv6, or a FQDN
type Addr struct {
	Unmarshaler

	FQDN string
	IP   net.IP
	Port int
}

func (a *Addr) Unmarshal(p []byte) error {
	if len(p) < 1 {
		return fmt.Errorf("unexpected addr buffer size")
	}

	atyp := p[0]
	p = p[1:]

	switch atyp {
	case AddrTypeIPV4:
		if len(p) < 4 {
			return fmt.Errorf("unexpected ipv4 size")
		}

		a.IP = net.IP(p[:4])
		p = p[4:]

	case AddrTypeIPV6:
		if len(p) < 16 {
			return fmt.Errorf("unexpected ipv6 size")
		}

		a.IP = net.IP(p[:16])
		p = p[16:]

	case AddrTypeFQDN:
		if len(p) < 1 {
			return fmt.Errorf("unable to retrieve fqdn addr length")
		}

		alen := int(p[0])
		p = p[1:]

		if len(p) < alen {
			return fmt.Errorf("unexpected fqdn addr size")
		}

		a.FQDN = string(p[:alen])
		p = p[alen:]

	default:
		return fmt.Errorf("unrecognized address type")
	}

	// Read the port
	if len(p) < 2 {
		return fmt.Errorf("unable to parse port")
	}
	a.Port = (int(p[0]) << 8) | int(p[1])

	return nil
}

func (a *Addr) String() string {
	if a.FQDN != "" {
		return fmt.Sprintf("%s (%s):%d", a.FQDN, a.IP, a.Port)
	}
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

// Address returns a string suitable to dial; prefer returning IP-based
// address, fallback to FQDN
func (a Addr) Address() string {
	if 0 != len(a.IP) {
		return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
	}
	return net.JoinHostPort(a.FQDN, strconv.Itoa(a.Port))
}
