package main

import (
	"log"
	"net"

	tproxy "github.com/elico/go-linux-tproxy"
	socks "github.com/fangdingjun/socks-go"
)

// var clientRegistry map[string]*tproxyDialer

// type TproxyDialer struct {
// 	ClientAdress string
// }

// func (TproxyDialer) New(ipAddress string) (*TproxyDialer, error) {
// 	switch {
// 	case govalidator.IsIP(ipAddress):
// 		return &TproxyDialer{ClientAdress: ipAddress}, errors.New("Only IPv4 and V6 addresses are supported")
// 	default:
// 		return &TproxyDialer{""}, errors.New("Only IPv4 and V6 addresses are supported")
// 	}
// }

// func (t *TproxyDialer) Dial(net, address string) (*net.Conn, error) {
// 	switch net {
// 	case "tcp":
// 		srvConn, err := tproxy.TcpDial(t.ClientAdress, address)
// 		return &srvConn, err
// 	default:
// 		return nil, errors.New("Only \"tcp\" net is supported")
// 	}
// }

func main() {

	// clientRegistry := make(map[string]*TproxyDialer, 100)

	conn, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Fatal(err)
	}

	for {
		c, err := conn.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("connected from %s", c.RemoteAddr())
		host, _, _ := net.SplitHostPort(c.RemoteAddr().String())
		// var tpDialer *TproxyDialer
		// var ok bool
		// if tpDialer, ok = clientRegistry[host]; ok {

		// }
		// t := tproxyDialer.new("192.168.89.1")
		// tpDialer.Dial("tcp", "192.168.89.5")

		// d := net.Dialer{Timeout: 10 * time.Second}

		s := socks.Conn{Conn: c, Dial: (func(net, address string) (net.Conn, error) {
			log.Println("Dialing a tproxy from:", host, "to:", address)
			srvConn, err := tproxy.TcpDial(host, address)
			if err != nil {
				log.Println("ERROR Dialing a tproxy from:", host, "to:", address, "ERROR:", err)
			}
			return srvConn, err
		}),
		}
		go s.Serve()
	}
}
