package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/elico/go-linux-tproxy"
)

var connRegistry = make(map[string]string)
var clientRegistry = make(map[string]http.Client)

func noRedirect(req *http.Request, via []*http.Request) error {
	return errors.New("Don't redirect!")
}
func getClient(srcipaddr string) http.Client {
	srcip, _, _ := net.SplitHostPort(srcipaddr)
	if _, ok := clientRegistry[srcip]; ok {
		return clientRegistry[srcip]
	}
	fakeIp := srcipaddr + ":0"
	var netTransport = &http.Transport{
		Dial: (func(network, addr string) (net.Conn, error) {
			// Resolve address
			//if the address is an IP
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			switch {
			case govalidator.IsIP(host):
				srvConn, err := tproxy.TCPDial(fakeIp, addr)
				if err != nil {
					return nil, err
				}
				return srvConn, nil
			case govalidator.IsDNSName(host):
				ips, err := net.LookupIP(host)
				if err != nil {
					return nil, err
				}
				for i, ip := range ips {
					srvConn, err := tproxy.TCPDial(fakeIp, net.JoinHostPort(ip.String(), port))
					if err != nil {
						fmt.Println(err)
						if i == len(ips) {
							return srvConn, nil
						}
						continue
					}
					fmt.Println("returning a srvconn")
					return srvConn, nil
				}
				srvConn, err := tproxy.TCPDial(fakeIp, addr)
				if err != nil {
					return nil, err
				}
				return srvConn, nil
			}
			return nil, nil
		}),
	}
	client := http.Client{Transport: netTransport, CheckRedirect: noRedirect}
	clientRegistry[srcip] = client
	return client
}

func main() {
	// init http server
	connRegistry = make(map[string]string)

	m := &MyHandler{}
	s := &http.Server{
		Handler: m,
	}

	// create custom listener
	nl, err := tproxy.TCPListen(":8080")

	//nl, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	l := &MyListener{nl}

	// serve through custom listener

	err = s.Serve(l)
	if err != nil {
		log.Fatal(err)
	}
}

// net.Conn

type MyConn struct {
	nc net.Conn
}

func (c MyConn) Read(b []byte) (n int, err error) {
	return c.nc.Read(b)
}

func (c MyConn) Write(b []byte) (n int, err error) {
	return c.nc.Write(b)
}

func (c MyConn) Close() error {
	delete(connRegistry, c.RemoteAddr().String())
	fmt.Println("Removed and Closing Conn from:", c.RemoteAddr().String(), "to:", c.LocalAddr().String())
	return c.nc.Close()
}

func (c MyConn) LocalAddr() net.Addr {
	return c.nc.LocalAddr()
}

func (c MyConn) RemoteAddr() net.Addr {
	return c.nc.RemoteAddr()
}

func (c MyConn) SetDeadline(t time.Time) error {
	return c.nc.SetDeadline(t)
}

func (c MyConn) SetReadDeadline(t time.Time) error {
	return c.nc.SetReadDeadline(t)
}

func (c MyConn) SetWriteDeadline(t time.Time) error {
	return c.nc.SetWriteDeadline(t)
}

// net.Listener

type MyListener struct {
	nl net.Listener
}

func (l MyListener) Accept() (c net.Conn, err error) {
	nc, err := l.nl.Accept()
	if err != nil {
		return nil, err
	}
	fmt.Println("Conn from:", nc.RemoteAddr().String(), "to:", nc.LocalAddr().String())

	connRegistry[nc.RemoteAddr().String()] = nc.LocalAddr().String()
	return MyConn{nc}, nil
}

func (l MyListener) Close() error {
	return l.nl.Close()
}

func (l MyListener) Addr() net.Addr {
	return l.nl.Addr()
}

// http.Handler

type MyHandler struct {
	// ...
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World\n", connRegistry[r.RemoteAddr], getClient(r.RemoteAddr), clientRegistry)

}
