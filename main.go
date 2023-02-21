package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/connection"
	"github.com/mhkarimi1383/pg_pro/logger"
	queryhelper "github.com/mhkarimi1383/pg_pro/query_helper"
)

func main() {
	defer logger.Sync()
	// Listen on a port for incoming connections
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", config.Get("listen_port")))
	if err != nil {
		logger.Panic(
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
			logger.Panic(
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
				logger.Panic(
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
			accessInfo, err := queryhelper.GetRelatedTables(msg.String)
			if err != nil {
				log.Printf("err: %+v\n", err)
				buf := (&pgproto3.ErrorResponse{
					Message: err.Error(),
				}).Encode(nil) // TODO: Add more fields
				_, err = conn.Write(buf)
				if err != nil {
					return errors.Wrap(err, "writing query error response")
				}
				buf = (&pgproto3.CommandComplete{}).Encode(nil)
				_, err = conn.Write(buf)
				if err != nil {
					return errors.Wrap(err, "writing CommandComplete response")
				}
				buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)
				_, err = conn.Write(buf)
				if err != nil {
					return errors.Wrap(err, "writing ReadyForQuery response")
				}
				continue
			}
			fmt.Printf("%+v\n", accessInfo)
			result, _ := connection.RunQuery(msg.String)

			buf := (&result.RowDescription).Encode(nil)
			_, err = conn.Write(buf)
			if err != nil {
				return errors.Wrap(err, "writing query response")
			}
			for _, d := range result.DataRows {
				buf = (&d).Encode(nil)
				_, err = conn.Write(buf)
				if err != nil {
					return errors.Wrap(err, "writing query response")
				}
			}
			if err != nil {
				return errors.Wrap(err, "writing query response")
			}
			buf = (&pgproto3.CommandComplete{}).Encode(nil)
			_, err = conn.Write(buf)
			if err != nil {
				return errors.Wrap(err, "writing query response")
			}
			buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)
			_, err = conn.Write(buf)
			if err != nil {
				return errors.Wrap(err, "writing query response")
			}
			log.Println(string(buf))
		case *pgproto3.Terminate:
			fmt.Println("Received terminate message")
			return nil
		default:
			return errors.Errorf("received unhandled message: %+v", msg)
		}
	}
}
