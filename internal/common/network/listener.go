package network

import (
	"net"
)

type QueueListener struct {
	queue <-chan *BufferedConn
	done  chan struct{}
}

func NewQueueListener(queue <-chan *BufferedConn) QueueListener {
	return QueueListener{
		queue: queue,
		done:  make(chan struct{}),
	}
}

func (l QueueListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.queue:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l QueueListener) Close() error {
	close(l.done)
	return nil
}

func (l QueueListener) Addr() net.Addr {
	return nil
}
