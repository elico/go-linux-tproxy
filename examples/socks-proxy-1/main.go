package main

import (
	"fmt"
	"log"
	"net"

	tproxy "github.com/elico/go-linux-tproxy"
	socks "github.com/fangdingjun/socks-go"
)

// https://gist.github.com/kotakanbe/d3059af990252ba89a82

func main() {
	tproxy.Debug = true
	conn, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Fatal(err)
	}
	ReadCidrListFile("cidr.txt")
	fmt.Println("Random hosts avaliable:", len(RandomHosts))
	if len(RandomHosts) == 0 {
		panic("0 Random host avaliable, the service must have at leat 1")
	}
	for {
		c, err := conn.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println("connected from ", c.RemoteAddr().String())

		host := RandomHosts[random(0, len(RandomHosts))] + ":0"
		// host := "10.0.3.1:0"
		s := socks.Conn{Conn: c, Dial: (func(net, address string) (net.Conn, error) {
			log.Println("Dialing a tproxy from:", host, "to:", address, "net:", net)
			srvConn, err := tproxy.TCPDial(host, address)
			if err != nil {
				log.Println("ERROR Dialing a tproxy from:", host, "to:", address, "ERROR:", err)
			}
			return srvConn, err
		})}

		go s.Serve()
	}
}
