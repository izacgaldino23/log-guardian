package unix

import "net"

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type Listener interface {
	Accept() (net.Conn, error)
	Close() error
}

type Conn interface {
	net.Conn
}

type ListenerFactory func(network, path string) (Listener, error)
