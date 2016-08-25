package main

import (
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/elico/go-linux-tproxy"
	"net"
	"net/http"
)

func noRedirect(req *http.Request, via []*http.Request) error {
	return errors.New("Don't redirect!")
}

func main() {
	fakeIp := "192.168.101.200:0"
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
				srvConn, err := tproxy.TcpDial(fakeIp, addr)
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
					srvConn, err := tproxy.TcpDial(fakeIp, net.JoinHostPort(ip.String(), port))
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
				srvConn, err := tproxy.TcpDial(fakeIp, addr)
				if err != nil {
					return nil, err
				}
				return srvConn, nil
			}
			return nil, nil
		}),
	}
	client := &http.Client{Transport: netTransport, CheckRedirect: noRedirect}
	resp, err := client.Get("http://www.google.com/")
	if err != nil && resp == nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)

}
