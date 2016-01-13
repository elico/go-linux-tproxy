package main

import (
	"net"
	"os"
	"fmt"
	"runtime"
	"bufio"
	"syscall"
	"flag"
	"io"
	"strings"
	"github.com/elico/go-linux-tproxy"
)
var tcp_port *string
var debug *bool


type bufferedConn struct {
	r        *bufio.Reader
	net.Conn // So that most methods are embedded
}

func newBufferedConn(c net.Conn) bufferedConn {
	return bufferedConn{bufio.NewReader(c), c}
}

func newBufferedConnSize(c net.Conn, n int) bufferedConn {
	return bufferedConn{bufio.NewReaderSize(c, n), c}
}

func (b bufferedConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func broker(dst, src bufferedConn, srcClosed chan struct{}) { 
	// We can handle errors in a finer-grained manner by inlining io.Copy (it's
	// simple, and we drop the ReaderFrom or WriterTo checks for
	// net.Conn->net.Conn transfers, which aren't needed). This would also let
	// us adjust buffersize.
	_, err := io.Copy(dst, src)
	// The next cases are very expected to happend in many cases that the client or server closes the connection.
	if err != nil && *debug {
		fmt.Fprintln(os.Stderr,"Copy error: %s", err)
	}
	if err := src.Close(); err != nil && *debug {
		fmt.Fprintln(os.Stderr,"Close error: %s", err)
	}
	src.Close()
	dst.Close()
	srcClosed <- struct{}{}
}

func handleConn(cliConn net.Conn, remoteAddr string) {
	serverClosed := make(chan struct{}, 1)
	clientClosed := make(chan struct{}, 1)
	
	// Here there should be a peek CODE which will allow to decide on what to do next with the connection.
	// This basic peeking code does something to most of the connections and stalls them slow enough to ge them stuck.

	fmt.Println("Before peeking")
	
	clientReaderBuff := newBufferedConn(cliConn)
	peeked , err := clientReaderBuff.Peek(256)
	fmt.Println(len(peeked))
	if strings.HasPrefix(string(peeked),"GET /") {
		fmt.Println("This is probably a HTTP connection")
	}
	if strings.HasPrefix(string(peeked),"GET http") {
		fmt.Println("This is probably a PROXY connection")
	}
	
	if 	strings.HasPrefix(string(peeked),"\x0D\x0A\x0D\x0A\x00\x0D\x0A\x51\x55\x49\x54\x0A") {
		fmt.Println("This is probably a HAPROXY PROXY_V2 connection")
	}
	
	fmt.Println("After peeking")

	srvConn, err := tproxy.TcpDial((cliConn.RemoteAddr()).String(), (cliConn.LocalAddr()).String())
	if srvConn == nil {
			if cliConn != nil {
				cliConn.Close()
			}
			if srvConn != nil {
				srvConn.Close()
			}
			if *debug {
				fmt.Fprintln(os.Stderr, "remote dial failed: %v\n", err)
			}
			return
	}
	serverReaderBuff := newBufferedConn(srvConn)
	if cliConn == nil {
		if *debug {
			fmt.Fprintln(os.Stderr,"copy(): oops, src is nil!")
		}
		if srvConn != nil {
			srvConn.Close()
		}
        return
    }
	if srvConn == nil {
		if *debug {
			fmt.Fprintln(os.Stderr,"copy(): oops, dst is nil!")
		}
		if cliConn != nil {
			cliConn.Close()
		}
        return
    }
	go broker(clientReaderBuff, serverReaderBuff, serverClosed)
	go broker(serverReaderBuff, clientReaderBuff, clientClosed)
	
	
	select {
	case <-clientClosed:
		tcp, ok := cliConn.(*net.TCPConn)
		if !ok {
			//fmt.Errorf("Bad conn type: %T", cliConn)
			_ = cliConn
		}
		_ = tcp
//		tcp.SetLinger(0)
//		tcp.CloseRead()
	case <-serverClosed:
		_ = srvConn
	}

	// Wait for the other connection to close.
	// This "waitFor" pattern isn't required, but gives us a way to track the
	// connection and ensure all copies terminate correctly; we can trigger
	// stats on entry and deferred exit of this function.
	//go io.Copy(local, remote)
	//go io.Copy(remote, local)
}


func main() {
    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        fmt.Println("Error Getting Rlimit ", err)
    }
    fmt.Print("Maximum FD per process: ")
    fmt.Println(rLimit)
    
	debug = flag.Bool("d", false, "Debug mode can be \"1\" for yes or \"0 for no")
	tcp_port = flag.String("l", ":9090", "ip:port for listening or plain \":port\" for listening all IPs")

	flag.Parse()

	l, err := tproxy.TcpListen(*tcp_port)
	if err != nil && *debug{
		fmt.Fprintln(os.Stderr, err)
	}
	for { 
		s, err := l.Accept()
		if err != nil {
		  //panic(err)
		  fmt.Println(err)
		}
		if *debug {
			fmt.Println("starting connection from: " + (s.RemoteAddr()).String() + ", to: " + (s.LocalAddr()).String())
		}
		go handleConn(s, (s.LocalAddr()).String())
	}
}
