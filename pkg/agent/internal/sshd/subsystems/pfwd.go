//go:build pfwd
// +build pfwd

package subsystems

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/lrita/cmap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	// {{if .Debug}}
	"log"
	// {{end}}
)

func init() {
	Subsystems["pfwd"] = subsystemPfwd
}

// subsystemPfwd implements subsystem for port forwarding
func subsystemPfwd(channel ssh.Channel, args []string) {
	defer channel.Close()

	if len(args) == 0 {
		channel.Write([]byte("Usage:\n\tlist\n\tstart <lport:rip:rport>\n\tstop <lport>\n"))
		return
	}

	switch args[0] {
	case "list":
		// list active port-forward sessions
		subsystemPfwdStorage.Range(func(key int, value *subsystemPfwdSession) bool {
			if _, err := channel.Write([]byte(value.Pretty() + "\n")); err != nil {
				return false
			}
			return true
		})
	case "start":
		// start new port-forward session
		if len(args) != 2 {
			channel.Write([]byte("Usage:\n\tstart <lport:rip:rport>\n"))
			return
		}
		// split argument
		splittedArg := strings.Split(args[1], ":")
		if len(splittedArg) != 3 {
			// {{if .Debug}}
			log.Printf("[pfwd] Malformed value to start port-forward: %s", args[1])
			// {{end}}
			channel.Write([]byte("Malformed value. Use format lport:rip:rport\n"))
			return
		}
		// get local port
		lport, err := strconv.Atoi(splittedArg[0])
		if err != nil {
			// {{if .Debug}}
			log.Printf("[pfwd] Parse local port for start: %s", err.Error())
			// {{end}}
			channel.Write([]byte(fmt.Sprintf("Parse local port: %s\n", err.Error())))
			return
		}
		// get remote ip
		rip := splittedArg[1]
		// get remote port
		rport, err := strconv.Atoi(splittedArg[2])
		if err != nil {
			// {{if .Debug}}
			log.Printf("[pfwd] Parse remote port for start: %s", err.Error())
			// {{end}}
			channel.Write([]byte(fmt.Sprintf("Parse remove port: %s\n", err.Error())))
			return
		}
		// create listener
		// TODO: remove harcoded 0.0.0.0
		l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", lport))
		if err != nil {
			// {{if .Debug}}
			log.Printf("[pfwd] Unable start listener on port %d: %s", lport, err.Error())
			// {{end}}
			channel.Write([]byte(fmt.Sprintf("Unable start listener on port %d: %s", lport, err.Error())))
		}
		// create session object
		v := &subsystemPfwdSession{
			localPort:  lport,
			remoteIp:   rip,
			remotePort: rport,
			listener:   l,
		}
		// start port-forward
		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				go v.handle(conn)
			}
		}()
		// save port-forward session
		subsystemPfwdStorage.Store(lport, v)
		// {{if .Debug}}
		log.Printf("[pfwd] Start port-forward session: %s", v.Pretty())
		// {{end}}
	case "stop":
		// stop existed port-forward session
		if len(args) != 2 {
			channel.Write([]byte("Usage:\n\tstop <lport>\n"))
			return
		}
		// get local port value
		lport, err := strconv.Atoi(args[1])
		if err != nil {
			// {{if .Debug}}
			log.Printf("[pfwd] Pase local port for stop: %s", err.Error())
			// {{end}}
			channel.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
			return
		}
		// load port-forward session
		if v, ok := subsystemPfwdStorage.Load(lport); ok {
			// remove session if such local port is busy
			if err := v.Stop(); err != nil {
				// {{if .Debug}}
				log.Printf("[pfwd] Stop listener: %s", err.Error())
				// {{end}}
				channel.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
			}
			subsystemPfwdStorage.Delete(lport)
			// {{if .Debug}}
			log.Printf("[pfwd] Stop port-forward session on %d", lport)
			// {{end}}
		}
	default:
		// {{if .Debug}}
		log.Printf("[pfwd] Unknown action %s", args[0])
		// {{end}}
		channel.Write([]byte("Unknown action. Choose from list, start, stop\n"))
		return
	}
}

// storage for active sessions
// key: local port
// value: port-forward session itself
var subsystemPfwdStorage cmap.Map[int, *subsystemPfwdSession]

type subsystemPfwdSession struct {
	localPort  int
	remoteIp   string
	remotePort int
	listener   net.Listener
}

// Pretty returns pretty string described port-forward session
func (s *subsystemPfwdSession) Pretty() string {
	// TODO: remove harcoded 0.0.0.0
	return fmt.Sprintf("0.0.0.0:%d -> %s:%d", s.localPort, s.remoteIp, s.remotePort)
}

// Stop stops listener for port-forward session
func (s *subsystemPfwdSession) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handle handle port-forward request
func (s *subsystemPfwdSession) handle(conn net.Conn) {
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	// create network TCP dialer
	dialer, err := net.Dial("tcp", net.JoinHostPort(s.remoteIp, strconv.Itoa(s.remotePort)))
	if err != nil {
		// {{if .Debug}}
		log.Printf("[pfwd] Unable dial connection to %s:%d: %s", s.remoteIp, s.remotePort, err.Error())
		// {{end}}
		return
	}
	defer func() {
		if dialer != nil {
			dialer.Close()
		}
	}()

	// start bi-directional traffic forward
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		if _, err := io.Copy(conn, dialer); err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		if _, err := io.Copy(dialer, conn); err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		// {{if .Debug}}
		log.Printf("[pfwd] Error on forwarding to %s:%d: %s", s.remoteIp, s.remotePort, err.Error())
		// {{end}}
		return
	}
	// {{if .Debug}}
	log.Printf("[pfwd] Stop forward session %s for client %s", s.Pretty(), conn.RemoteAddr())
	// {{end}}
}
