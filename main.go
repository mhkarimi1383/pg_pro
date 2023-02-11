package main

import (
	"fmt"
	"io"
	"net"

	cowsay "github.com/Code-Hex/Neo-cowsay/v2"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/mhkarimi1383/pg_pro/config"
	queryhelper "github.com/mhkarimi1383/pg_pro/query_helper"
)

var (
	logger *zap.Logger
)

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func main() {
	defer logger.Sync()
	// Listen on a port for incoming connections
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", config.Get("listen_port")))
	if err != nil {
		logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Fatal(
			err.Error(),
			zap.String("event", "listen"),
			zap.Uint("port", config.GetUint("listen_port")),
		)
	}
	defer ln.Close()

	logger.Info(
		"listener started",
		zap.String("event", "listen"),
		zap.Uint("port", config.GetUint("listen_port")),
	)
	// Accept incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Fatal(
				err.Error(),
				zap.String("event", "accept"),
				zap.Uint("port", config.GetUint("listen_port")),
			)
		}

		go func() {
			err := handleConnection(conn)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				logger.Info(
					"EOF error possibly a client disconected unexpectedly",
					zap.String("event", "handle connection"),
				)
			} else if err != nil {
				logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Fatal(
					err.Error(),
					zap.String("event", "handle connection"),
				)
			}
		}()
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	backend := pgproto3.NewBackend(conn, conn)

	// Read the initial startup message
	startupMsg, err := backend.ReceiveStartupMessage()
	if err != nil {
		return err
	}

	fmt.Printf("Received startup message: %+v\n", startupMsg)
	backend.SetAuthType(pgproto3.AuthTypeMD5Password)
	switch startupMsg.(type) {
	case *pgproto3.StartupMessage:
		buf := (&pgproto3.AuthenticationMD5Password{}).Encode(nil)
		_, err = conn.Write(buf)
		if err != nil {
			return errors.Wrap(err, "sending AuthenticationMD5Password to client")
		}
		msg, err := backend.Receive()
		if err != nil {
			return err
		}
		msgPass := msg.(*pgproto3.PasswordMessage)
		fmt.Printf("entered password: %v, %v\n", msgPass.Password, err)
		buf = (&pgproto3.AuthenticationOk{}).Encode(nil)
		_, err = conn.Write(buf)
		if err != nil {
			return errors.Wrap(err, "sending AuthenticationOk to client")
		}
		buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)
		_, err = conn.Write(buf)
		if err != nil {
			return errors.Wrap(err, "sending ready for query to client")
		}
	case *pgproto3.SSLRequest:
		_, err = conn.Write([]byte("N"))
		if err != nil {
			return errors.Wrap(err, "sending deny SSL request to client")
		}
		return handleConnection(conn)
	default:
		return errors.Errorf("unknown startup message: %#v", startupMsg)
	}

	fmt.Println("user logged in")
	// Read and handle incoming messages
	for {
		msg, err := backend.Receive()
		if err != nil {
			return errors.Wrap(err, "receive client query")
		}

		switch msg := msg.(type) {
		case *pgproto3.Query:
			fmt.Printf("Received query: %s\n", msg.String)
			isRead, _ := queryhelper.IsReadOperation(msg.String)
			// if err != nil {
			// 	return err // TODO: check if that was a user mistake or not (do not make postgres to handle user mistakes)
			// }
			say, err := cowsay.Say(
				fmt.Sprintf(`Your query was
"%v"
but I am not ready yet
ReadOperation: "%v"`, msg.String, isRead),
				cowsay.Type("elephant"),
			)
			if err != nil {
				return errors.Wrap(err, "generating query response")
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
				return errors.Wrap(err, "writing query response")
			}
		case *pgproto3.Terminate:
			fmt.Println("Received terminate message")
			return nil
		default:
			return errors.Errorf("received unhandled message: %+v", msg)
		}
	}
}
