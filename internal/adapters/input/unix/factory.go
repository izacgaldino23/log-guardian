package unix

import (
	"net"
	"time"
)

type connectionProvider struct{}

func (c connectionProvider) DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

func NewUnixConnectionProvider() ConnectionProvider {
	return connectionProvider{}
}
