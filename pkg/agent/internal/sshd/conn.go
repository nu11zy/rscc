package sshd

import (
	"agent/internal/config"
	"agent/internal/logger"
	"math/rand/v2"
	"net"
	"time"
)

const (
	connectDelay = 2
	shuffleDelay = 5
	connTimeout  = 10
	retries      = 10
)

func NewTCPConn() (net.Conn, string) {
	lg := logger.GetLogger()

	for i := 0; i < retries; i++ {
		servers := config.GetServers()
		rand.Shuffle(len(servers), func(i, j int) {
			servers[i], servers[j] = servers[j], servers[i]
		})

		for _, server := range servers {
			lg.Info("Trying to connect to %s", server)
			conn, err := net.Dial("tcp", server)
			if err != nil {
				time.Sleep(time.Duration(connTimeout+rand.IntN(connTimeout)) * time.Second)
				continue
			}
			return conn, server
		}
		time.Sleep(time.Duration(shuffleDelay+rand.IntN(shuffleDelay)) * time.Second)
	}

	lg.Error("connection error")
	return nil, ""
}
