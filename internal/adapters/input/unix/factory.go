package unix

import (
	"time"
)

func NewNetConnectionFactory(newNetConnection newNetConnection) ConnectionFactory {
	return func(network, address string, timeout time.Duration) (Conn, error) {
		conn, err := newNetConnection(network, address, timeout)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}
