package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/elico/go-linux-tproxy"
)

func main() {
	tproxy.Debug = true
	fakeIp := "10.0.2.253:999"
	dst := "192.168.89.1:80"

	// fmt.Println(govalidator.IsIP(host))
	// srvConn, err := tproxy.TCPDial(fakeIp, dst)
	// if err != nil {
	// 	log.Println(err)
	// }
	// log.Println(srvConn)

	log.Println("Dialing a tproxy from:", fakeIp, "to:", dst)
	srvConn, err := tproxy.TCPDial(fakeIp, dst)
	if err != nil {
		log.Println("ERROR Dialing a tproxy from:", fakeIp, "to:", dst, "ERROR:", err)
	}
	srvConn.Write([]byte("GET / HTTP/1.1\r\nHost: 192.168.89.1\r\nConnection: close\r\n\r\n"))
	b, err := ioutil.ReadAll(srvConn)
	if err != nil {
		log.Println("ERROR Recieving a tproxy from:", fakeIp, "to:", dst, "ERROR:", err)
	}
	fmt.Println(string(b))

}
