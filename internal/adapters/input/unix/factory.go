package unix

import (
	"net"
	"time"
)

func NewNetConnectionFactory() ConnectionFactory {
	return func(network, address string, timeout time.Duration) (Conn, error) {
		conn, err := net.DialTimeout(network, address, timeout)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}
