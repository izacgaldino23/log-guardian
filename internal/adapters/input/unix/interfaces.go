package unix

import (
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type Conn interface {
	Close() error
	SetReadDeadline(t time.Time) error
	Read(b []byte) (n int, err error)
}

type ConnectionProvider interface {
	DialTimeout(network, address string, timeout time.Duration) (Conn, error)
}
