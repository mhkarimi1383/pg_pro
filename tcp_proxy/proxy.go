package tcpproxy

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"

	"github.com/mhkarimi1383/pg_pro/config"
)

func proxyConn(conn *net.TCPConn) {
	rAddrStr := ""
	sources := config.GetSlice("sources")
	for _, source := range sources {
		src := source.(map[string]any)
		if src["mode"] == "master" {
			addr, err := url.Parse(fmt.Sprintf("%v", src["url"]))
			if err != nil {
				panic(err)
			}
			rAddrStr = addr.Host
		}
	}

	rAddr, err := net.ResolveTCPAddr("tcp", rAddrStr)
	if err != nil {
		panic(err)
	}

	rConn, err := net.DialTCP("tcp", nil, rAddr)
	if err != nil {
		panic(err)
	}
	defer rConn.Close()

	buf := &bytes.Buffer{}
	for {
		data := make([]byte, 256)
		var n int
		n, err = conn.Read(data)
		if err != nil {
			panic(err)
		}
		buf.Write(data[:n])
		if data[0] == 13 && data[1] == 10 {
			break
		}
	}

	if _, err = rConn.Write(buf.Bytes()); err != nil {
		panic(err)
	}
	log.Printf("sent:\n%v", hex.Dump(buf.Bytes()))

	data := make([]byte, 1024)
	n, err := rConn.Read(data)
	if err != nil {
		if err != io.EOF {
			panic(err)
		} else {
			log.Printf("received err: %v", err)
		}
	}
	log.Printf("received:\n%v", hex.Dump(data[:n]))
}

func handleConn(in <-chan *net.TCPConn, out chan<- *net.TCPConn) {
	for conn := range in {
		proxyConn(conn)
		out <- conn
	}
}

func closeConn(in <-chan *net.TCPConn) {
	for conn := range in {
		conn.Close()
	}
}

func Serve() error {
	addr, err := net.ResolveTCPAddr("tcp", ":" + config.GetString("listen_port"))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	pending, complete := make(chan *net.TCPConn), make(chan *net.TCPConn)

	for i := 0; i < 5; i++ {
		go handleConn(pending, complete)
	}
	go closeConn(complete)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			return err
		}
		pending <- conn
	}
}

