package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
)

func main() {
	local := NewNetwork()
	local.ListIPsAndPorts()
}

// NewNetwork starts scanning the local ipaddress of the machine
// returns a local ip view of the machine
func NewNetwork() *Network {

	nw := &Network{}
	nw.IPsAndPorts = make(map[string][]int)
	nw.getLocalIPAddresses()

	for _, address := range nw.MyIPAddresses {
		ports, err := nw.ScanHost(address)
		if err != nil {
			fmt.Println(err)
		}
		nw.IPsAndPorts[address] = ports
	}

	return nw
}

// Network ...
// MyIPAddresses is a list of local machine IPs
// IPsAndPorts map the open ports onto each IP
type Network struct {
	MyIPAddresses []string
	IPsAndPorts   map[string][]int
}

// ListIPsAndPorts prints out the found local IPs and their open Ports
func (network *Network) ListIPsAndPorts() {
	printerCyanUnderline := color.New(color.FgCyan)
	printerGreen := color.New(color.FgGreen)

	if len(network.IPsAndPorts) == 0 {
		color.Red("No IP addresses found")
	}
	for ip, ports := range network.IPsAndPorts {
		// fmt.Println(ip, ports)
		print("IP: ")
		printerCyanUnderline.Print(ip + " ")

		for _, v := range ports {
			printerGreen.Print(v, " ")
		}
		fmt.Println("")
	}

}

// getLocalIPAddresses looks at the local machine to establish
// which ipaddresses are bound to the network
func (network *Network) getLocalIPAddresses() error {
	var ipAddresses []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return errors.New("No network interfaces available")
	}

	for _, interf := range interfaces {

		if interf.Flags&net.FlagUp == 0 {
			continue
		}

		if interf.Flags&net.FlagLoopback != 0 {
			continue
		}

		addresses, err := interf.Addrs()
		if err != nil {
			return errors.New("Couldn't fetch inface addresses")
		}
		for _, address := range addresses {

			var ip net.IP

			switch a := address.(type) {
			case *net.IPNet:
				ip = a.IP
			case *net.IPAddr:
				ip = a.IP
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			ipAddresses = append(ipAddresses, ip.String())
		}
	}

	if len(ipAddresses) == 0 {
		return errors.New("No ipAddresses available")
	}
	network.MyIPAddresses = ipAddresses
	return nil
}

// ScanHost scans the given host/ipaddress from port 0 - 65536
// with a default timeout of 2 seconds
// Returns []int ports
func (network *Network) ScanHost(host string) ([]int, error) {
	ports, err := network.TCPScanner(host, 0, 65535, 2*time.Second)
	if err != nil {
		return nil, err
	}
	return ports, nil
}

// TCPScanner ...
// Ports to scan can be from 0 to 65535
func (network *Network) TCPScanner(host string, startPort int, endPort int, timeout time.Duration) ([]int, error) {

	if host == "" {
		return nil, errors.New("No host to search")
	}

	if endPort == 0 {
		log.Println("endPort not set, defaulting to 65535")
		endPort = 65535
	}

	ports := []int{}
	waitGroup := &sync.WaitGroup{}
	mutex := sync.Mutex{}
	for port := startPort; port <= endPort; port++ {
		waitGroup.Add(1)

		go func(host string, port int, timeout time.Duration) {
			if network.OpenPort(host, port, timeout) {
				mutex.Lock()
				ports = append(ports, port)
				mutex.Unlock()
			}
			waitGroup.Done()
		}(host, port, timeout)
	}
	waitGroup.Wait()
	return ports, nil
}

// OpenPort opens the port on host:port with a timeout
// if opened wil return true
// else false
func (network *Network) OpenPort(host string, port int, timeout time.Duration) bool {
	connString := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", connString, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
