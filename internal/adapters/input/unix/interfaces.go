package unix

import (
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type Conn interface {
	Close() error
	SetReadDeadline(time.Time) error
	Read(p []byte) (n int, err error)
}

type newNetConnection func(network, address string, timeout time.Duration) (Conn, error)

type ConnectionFactory func(network, address string, timeout time.Duration) (Conn, error)
