// Package tproxy provides the TcpDial and TcpListen tproxy equivalent of the
// net package Dial and Listen with tproxy support for linux ONLY.
package tproxy

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

const big = 0xFFFFFF
const IP_ORIGADDRS = 20

// Debug outs the library in Debug mode
var Debug = false

func ipToSocksAddr(family int, ip net.IP, port int, zone string) (unix.Sockaddr, error) {
	switch family {
	case unix.AF_INET:
		if len(ip) == 0 {
			ip = net.IPv4zero
		}
		if ip = ip.To4(); ip == nil {
			return nil, net.InvalidAddrError("non-IPv4 address")
		}
		sa := new(unix.SockaddrInet4)
		for i := 0; i < net.IPv4len; i++ {
			sa.Addr[i] = ip[i]
		}
		sa.Port = port
		return sa, nil
	case unix.AF_INET6:
		if len(ip) == 0 {
			ip = net.IPv6zero
		}
		// IPv4 callers use 0.0.0.0 to mean "announce on any available address".
		// In IPv6 mode, Linux treats that as meaning "announce on 0.0.0.0",
		// which it refuses to do.  Rewrite to the IPv6 unspecified address.
		if ip.Equal(net.IPv4zero) {
			ip = net.IPv6zero
		}
		if ip = ip.To16(); ip == nil {
			return nil, net.InvalidAddrError("non-IPv6 address")
		}
		sa := new(unix.SockaddrInet6)
		for i := 0; i < net.IPv6len; i++ {
			sa.Addr[i] = ip[i]
		}
		sa.Port = port
		sa.ZoneId = uint32(zoneToInt(zone))
		return sa, nil
	}
	return nil, net.InvalidAddrError("unexpected socket family")
}

func zoneToInt(zone string) int {
	if zone == "" {
		return 0
	}
	if ifi, err := net.InterfaceByName(zone); err == nil {
		return ifi.Index
	}
	n, _, _ := dtoi(zone, 0)
	return n
}

func dtoi(s string, i0 int) (n int, i int, ok bool) {
	n = 0
	for i = i0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return 0, i, false
		}
	}
	if i == i0 {
		return 0, i, false
	}
	return n, i, true
}

// IPv6TcpAddrToUnixSocksAddr ---
func IPv6TcpAddrToUnixSocksAddr(addr string) (sa unix.Sockaddr, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp6", addr)
	if err != nil {
		return nil, err
	}
	return ipToSocksAddr(unix.AF_INET6, tcpAddr.IP, tcpAddr.Port, tcpAddr.Zone)
}

// IPv6UdpAddrToUnixSocksAddr ---
func IPv6UdpAddrToUnixSocksAddr(addr string) (sa unix.Sockaddr, err error) {
	tcpAddr, err := net.ResolveTCPAddr("udp6", addr)
	if err != nil {
		return nil, err
	}
	return ipToSocksAddr(unix.AF_INET6, tcpAddr.IP, tcpAddr.Port, tcpAddr.Zone)
}

// TCPListen is listening for incoming IP packets which are being intercepted.
// In conflict to regular Listen mehtod the socket destination and source addresses
// are of the intercepted connection.
// Else then that it works exactly like net package net.Listen.
func TCPListen(listenAddr string) (listener net.Listener, err error) {
	s, err := unix.Socket(unix.AF_INET6, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(s)
	err = unix.SetsockoptInt(s, unix.SOL_IP, unix.IP_TRANSPARENT, 1)
	if err != nil {
		return nil, err
	}

	sa, err := IPv6TcpAddrToUnixSocksAddr(listenAddr)
	if err != nil {
		return nil, err
	}
	err = unix.Bind(s, sa)
	if err != nil {
		return nil, err
	}
	err = unix.Listen(s, unix.SOMAXCONN)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(s), "TProxy")
	defer f.Close()
	return net.FileListener(f)
}

// TCPDial is a special tcp connection which binds a non local address as the source.
// Except then the option to bind to a specific local address which the machine doesn't posses
// it is exactly like any other net.Conn connection.
// It is advised to use port numbered 0 in the localAddr and leave the kernel to choose which
// Local port to use in order to avoid errors and binding conflicts.
func TCPDial(localAddr, remoteAddr string) (conn net.Conn, err error) {
	if Debug {
		fmt.Println("TcpDial from:", localAddr, "to:", remoteAddr)
	}
	s, err := unix.Socket(unix.AF_INET6, unix.SOCK_STREAM, 0)

	//In a case there was a need for a non-blocking socket an example
	//s, err := unix.Socket(unix.AF_INET6, unix.SOCK_STREAM |unix.SOCK_NONBLOCK, 0)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer unix.Close(s)
	err = unix.SetsockoptInt(s, unix.SOL_IP, unix.IP_TRANSPARENT, 1)
	if err != nil {
		if Debug {
			fmt.Println("ERROR setting the socket in IP_TRANSPARENT mode", err)
		}

		return nil, err
	}

	err = unix.SetsockoptInt(s, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		if Debug {
			fmt.Println("ERROR setting the socket in unix.SO_REUSEADDR mode", err)
		}
		return nil, err
	}

	rhost, _, err := net.SplitHostPort(localAddr)
	if err != nil {
		if Debug {
			// fmt.Fprintln(os.Stderr, err)
			fmt.Println("ERROR", err, "running net.SplitHostPort on address:", localAddr)
		}
	}

	sa, err := IPv6TcpAddrToUnixSocksAddr(rhost + ":0")
	if err != nil {
		if Debug {
			fmt.Println("ERROR creating a hostaddres for the socker with IPv6TcpAddrToUnixSocksAddr", err)
		}
		return nil, err
	}

	remoteSocket, err := IPv6TcpAddrToUnixSocksAddr(remoteAddr)
	if err != nil {
		if Debug {
			fmt.Println("ERROR creating a remoteSocket for the socker with IPv6TcpAddrToUnixSocksAddr on the remote addres", err)
		}
		return nil, err
	}

	err = unix.Bind(s, sa)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = unix.Connect(s, remoteSocket)
	if err != nil {
		if Debug {
			fmt.Println("ERROR Connecting from", s, "to:", remoteSocket, "ERROR:", err)
		}
		return nil, err
	}

	f := os.NewFile(uintptr(s), "TProxyTCPClient")
	client, err := net.FileConn(f)
	if err != nil {
		if Debug {
			fmt.Println("ERROR os.NewFile", err)
		}
		return nil, err
	}
	if Debug {
		fmt.Println("FINISHED Creating net.coo from:", client.LocalAddr().String(), "to:", client.RemoteAddr().String())
	}
	return client, err
}
