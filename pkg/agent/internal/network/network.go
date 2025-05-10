package network

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"
)

func NewTCPConn(ctx context.Context, servers []string) (net.Conn, string, error) {
	// Select random server from servers and try to connect to it.
	shuffleSleep := 5
	connectSleep := 2
	for i := 0; i < 10; i++ {
		shuffledServers := servers
		rand.Shuffle(len(shuffledServers), func(i, j int) {
			shuffledServers[i], shuffledServers[j] = shuffledServers[j], shuffledServers[i]
		})
		for _, server := range shuffledServers {
			conn, err := net.Dial("tcp", server)
			if err != nil {
				time.Sleep(time.Duration(connectSleep*i) * time.Second)
				continue
			}
			return conn, server, nil
		}
		time.Sleep(time.Duration(shuffleSleep*i) * time.Second)
	}

	return nil, "", fmt.Errorf("connection error")
}
