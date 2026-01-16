package unix

import (
	"net"
	"time"
)

type connectionProvider struct {}

func NewUnixConnectionProvider(address string, timeout time.Duration) (Conn, error) {
	return net.DialTimeout("unix", address, timeout)
}
