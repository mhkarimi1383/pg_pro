package main

import (
	"fmt"
	"log"
	"net"

	cowsay "github.com/Code-Hex/Neo-cowsay/v2"
	"github.com/jackc/pgx/v5/pgproto3"
)

func main() {
	// Listen on a port for incoming connections
	ln, err := net.Listen("tcp", ":5432")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	fmt.Println("Listening on :5432...")

	// Accept incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	backend := pgproto3.NewBackend(conn, conn)

	// Read the initial startup message
	startupMsg, err := backend.ReceiveStartupMessage()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Received startup message: %+v\n", startupMsg)
	backend.SetAuthType(pgproto3.AuthTypeMD5Password)
	switch startupMsg.(type) {
	case *pgproto3.StartupMessage:
		log.Printf("startupMsg.(*pgproto3.StartupMessage): %+v\n", startupMsg.(*pgproto3.StartupMessage))
		buf := (&pgproto3.AuthenticationMD5Password{}).Encode(nil)
		fmt.Printf("buf: %v\n", string(buf))
		_, err = conn.Write(buf)
		if err != nil {
			log.Fatalf("error sending ready for query: %v", err)
		}
		msg, err := backend.Receive()
		msgPass := msg.(*pgproto3.PasswordMessage)
		fmt.Printf("startupMsg: %v, %v\n", msgPass.Password, err)
		buf = (&pgproto3.AuthenticationOk{}).Encode(nil)
		fmt.Printf("buf: %v\n", string(buf))
		_, err = conn.Write(buf)
		if err != nil {
			log.Fatalf("error sending ready for query: %v", err)
		}
		buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)
		_, err = conn.Write(buf)
		if err != nil {
			log.Fatalf("error sending ready for query: %v", err)
		}
	case *pgproto3.SSLRequest:
		_, err = conn.Write([]byte("N"))
		if err != nil {
			log.Fatalf("error sending deny SSL request: %v", err)
		}
		handleConnection(conn)
		return
	default:
		log.Fatalf("unknown startup message: %#v", startupMsg)
	}

	fmt.Println("Sent AuthenticationOk message")
	// Read and handle incoming messages
	for {
		msg, err := backend.Receive()
		if err != nil {
			panic(err)
		}

		switch msg := msg.(type) {
		case *pgproto3.Query:
			fmt.Printf("Received query: %s\n", msg.String)
			if err != nil {
				log.Fatalf("error generating query response: %v", err)
			}
			say, err := cowsay.Say(
				fmt.Sprintf(`Your query was
'%v'
but I am not ready yet`, msg.String),
				cowsay.Type("elephant"),
			)
			if err != nil {
				log.Fatalf("error generating query response: %v", err)
			}
			buf := (&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{
				{
					Name:                 []byte("Elephant Answer"),
					TableOID:             0,
					TableAttributeNumber: 0,
					DataTypeOID:          25,
					DataTypeSize:         -1,
					TypeModifier:         -1,
					Format:               0,
				},
			}}).Encode(nil)
			buf = (&pgproto3.DataRow{Values: [][]byte{[]byte(say)}}).Encode(buf)
			buf = (&pgproto3.CommandComplete{}).Encode(buf)
			buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf)
			_, err = conn.Write(buf)
			if err != nil {
				log.Fatalf("error writing query response: %v", err)
			}
		case *pgproto3.Terminate:
			fmt.Println("Received terminate message")
			return
		default:
			fmt.Printf("Received unhandled message: %+v\n", msg)
		}
	}
}
