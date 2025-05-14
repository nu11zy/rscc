//go:build pscan
// +build pscan

package subsystems

import (
	"context"
	"flag"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/malfunkt/iprange"

	"golang.org/x/crypto/ssh"

	// {{if .Debug}}
	"log"
	// {{end}}
)

func init() {
	Subsystems["pscan"] = subsystemPscan
}

// subsystemPscan implements subsystem for network TCP scanning
func subsystemPscan(channel ssh.Channel, args []string) {
	defer channel.Close()

	// hold raw string with IPs to scan
	var cmdIps string
	// hold raw string with ports to scan
	var cmdPorts string
	// hold raw int value for timeout in case of TCP connection
	var cmdTimeout int
	// hold raw int valut for number of scanner's threads
	var cmdThreads int

	// parse arguments
	subsystemCommandline := &flag.FlagSet{}
	subsystemCommandline.SetOutput(channel)
	subsystemCommandline.StringVar(&cmdIps, "ips", "", "IP addresses to scan (required)")
	subsystemCommandline.StringVar(&cmdPorts, "ports", "21,22,23,25,53,80,88,102,161,162,389,443,445,636,1433,3128,1962,3389,4786,5985,5986,7433,8080-8200,9000-9200,9433,9600,10000,10161,10162", "Ports to scan")
	subsystemCommandline.IntVar(&cmdThreads, "threads", 300, "Number of threads for scanner")
	subsystemCommandline.IntVar(&cmdTimeout, "timeout", 3, "Timeout for TCP connection establishment")
	if len(args) == 0 {
		channel.Write([]byte("Usage:\n"))
		subsystemCommandline.PrintDefaults()
		return
	}
	subsystemCommandline.Parse(args)

	// validate arguments
	if cmdIps == "" {
		// {{if .Debug}}
		log.Println("[scan] Unspecified or blank --ips flag")
		// {{end}}
		channel.Write([]byte("[scan] Unspecified or blank --ips flag\n"))
		return
	}
	if cmdThreads <= 0 {
		// {{if .Debug}}
		log.Println("[scan] Invalid value for flag --threads")
		// {{end}}
		channel.Write([]byte("[scan] Invalid value for flag --threads\n"))
		return
	}
	if cmdTimeout <= 0 {
		// {{if .Debug}}
		log.Println("[scan] Invalid value for flag --timeout")
		// {{end}}
		channel.Write([]byte("[scan] Invalid value for flag --timeout\n"))
		return
	}

	// parse IP addresses
	ips := parseIps(cmdIps)
	slices.Sort(ips)
	ips = slices.Compact(ips)
	if len(ips) == 0 {
		// {{if .Debug}}
		log.Println("[scan] No IPs to scan")
		// {{end}}
		channel.Write([]byte("[scan] No IPs to scan\n"))
		return
	}

	// parse ports
	ports := parsePorts(cmdPorts)
	slices.Sort(ports)
	ports = slices.Compact(ports)
	if len(ports) == 0 {
		// {{if .Debug}}
		log.Println("[scan] No ports to scan")
		// {{end}}
		channel.Write([]byte("[scan] No ports to scan\n"))
		return
	}

	// prepare config for scanner
	config := &subsystemPscanConfig{
		ips:     ips,
		ports:   ports,
		timeout: cmdTimeout,
		threads: cmdThreads,
		ch:      channel,
	}

	// context to control scanner lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			bytes := make([]byte, 1)
			if _, err := channel.Read(bytes); err != nil {
				cancel()
				return
			}
		}
	}()
	config.Scan(ctx)
}

type subsystemPscanConfig struct {
	ips     []string
	ports   []int
	timeout int
	threads int
	ch      ssh.Channel
}

type subsystemPscanAddress struct {
	ip   string
	port int
}

// Pretty returns address in pretty format ip:port
func (s *subsystemPscanAddress) Pretty() string {
	return fmt.Sprintf("%s:%d", s.ip, s.port)
}

// Scan start TCP scanning
func (s *subsystemPscanConfig) Scan(ctx context.Context) {
	addrs := make(chan subsystemPscanAddress, 65535)
	var wg sync.WaitGroup

	// {{if .Debug}}
	log.Printf("[scan] Start new scan on ips %v with ports %v on %d threads", s.ips, s.ports, s.threads)
	// {{end}}

	for range s.threads {
		go func() {
			for addr := range addrs {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				default:
					s.connectTcp(ctx, addr)
					wg.Done()
				}
			}
		}()
	}

	// fill runtime chan with combination
	for _, port := range s.ports {
		for _, ip := range s.ips {
			wg.Add(1)
			addrs <- subsystemPscanAddress{ip, port}
		}
	}

	wg.Wait()
	close(addrs)
}

// connectTcp try to connect to ip:port
func (s *subsystemPscanConfig) connectTcp(ctx context.Context, addr subsystemPscanAddress) {
	// dialer with controllable timeout
	dialer := &net.Dialer{
		Timeout: time.Duration(s.timeout) * time.Second,
	}
	// connect to ip:port
	conn, err := dialer.DialContext(ctx, "tcp4", addr.Pretty())
	if err == nil {
		// print open port
		// {{if .Debug}}
		log.Printf("[scan] %s\n", addr.Pretty())
		// {{end}}
		s.ch.Write([]byte(fmt.Sprintf("%s\n", addr.Pretty())))
		if conn != nil {
			conn.Close()
		}
	}
}

func parseIps(raw string) []string {
	var res []string

	if strings.Contains(raw, ",") {
		rawHostList := strings.Split(raw, ",")
		var hosts []string
		for _, rawHost := range rawHostList {
			hosts = ipParser(rawHost)
			res = append(res, hosts...)
		}
	} else {
		res = ipParser(raw)
	}

	return res
}

func ipParser(raw string) []string {
	switch {
	case strings.Contains(raw, "/"):
		hosts := rangeParser(raw)
		if len(hosts) < 2 {
			return hosts
		}
		return hosts[1 : len(hosts)-1]
	case strings.Contains(raw, "-"):
		return rangeParser(raw)
	default:
		host := net.ParseIP(raw)
		if host != nil {
			return []string{host.String()}
		}
	}
	return nil
}

func rangeParser(raw string) []string {
	var hosts []string

	netHosts, err := iprange.ParseList(raw)
	if err != nil {
		return nil
	}

	for _, host := range netHosts.Expand() {
		hosts = append(hosts, host.String())
	}

	return hosts
}

func parsePorts(raw string) []int {
	var ports []int
	if raw == "" {
		return nil
	}

	slices := strings.Split(raw, ",")
	for _, port := range slices {
		port = strings.TrimSpace(port)
		if port == "" {
			continue
		}

		if strings.Contains(port, "-") {
			ranges := strings.Split(port, "-")
			if len(ranges) < 2 {
				continue
			}

			startPort, err := strconv.Atoi(ranges[0])
			if err != nil {
				continue
			}
			if !isPortValid(startPort) {
				continue
			}

			endPort, err := strconv.Atoi(ranges[1])
			if err != nil {
				continue
			}
			if !isPortValid(endPort) {
				continue
			}

			if startPort < endPort {
				for i := startPort; i <= endPort; i++ {
					ports = append(ports, i)
				}
			} else {
				for i := endPort; i <= startPort; i++ {
					ports = append(ports, i)
				}
			}
		} else {
			startPort, err := strconv.Atoi(port)
			if err != nil {
				continue
			}
			if !isPortValid(startPort) {
				continue
			}
			ports = append(ports, startPort)
		}
	}
	return ports
}

func isPortValid(port int) bool {
	return (port <= 65535 && port >= 1)
}
