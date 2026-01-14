package unix

import "net"

func NewNetListenerFactory() ListenerFactory {
	return func(network, path string) (Listener, error) {
		listener, err := net.Listen(network, path)
		if err != nil {
			return nil, err
		}
		return listener, nil
	}
}
