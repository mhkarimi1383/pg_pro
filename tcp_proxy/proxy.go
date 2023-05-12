package tcpproxy

import (
	"log"
	"net"

	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/connection"
)

func Serve() error {
	addr, err := net.ResolveTCPAddr("tcp", ":"+config.GetString("listen_port"))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	log.Printf("Proxy server started on port %v", config.GetString("listen_port"))

	for {
		clientConn, err := listener.AcceptTCP()
		if err != nil {
			return err
		}

		go handleClient(clientConn)
	}
}

func handleClient(cliConn *net.TCPConn) {
	defer cliConn.Close()

	serverConn, err := connection.GetRawConnection()
	if err != nil {
		log.Printf("Failed to connect to server: %s", err)
		return
	}
	defer serverConn.Close()

	Proxy(serverConn.(*net.TCPConn), cliConn)
}

func Proxy(srvConn, cliConn *net.TCPConn) {
	serverClosed := make(chan struct{}, 1)
	clientClosed := make(chan struct{}, 1)

	go broker(srvConn, cliConn, clientClosed)
	go broker(cliConn, srvConn, serverClosed)

	var waitFor chan struct{}
	select {
	case <-clientClosed:
		srvConn.SetLinger(0)
		srvConn.CloseRead()
		waitFor = serverClosed
	case <-serverClosed:
		cliConn.CloseRead()
		waitFor = clientClosed
	}

	<-waitFor
}

func broker(dst, src net.Conn, srcClosed chan struct{}) {
	_, err := copyBuffer(dst, src, nil)
	if err != nil {
		log.Printf("Copy error: %s", err)
	}
	if err := src.Close(); err != nil {
		log.Printf("Close error: %s", err)
	}
	srcClosed <- struct{}{}
}