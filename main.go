package main

import (
	"fmt"
	"io"
	"net"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	pg_query "github.com/pganalyze/pg_query_go/v4/parser"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/mhkarimi1383/pg_pro/auth"
	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/connection"
	"github.com/mhkarimi1383/pg_pro/logger"
	queryhelper "github.com/mhkarimi1383/pg_pro/query_helper"
	"github.com/mhkarimi1383/pg_pro/types"
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

	logger.Info(
		"received startup message",
		zap.String("event", "new connection"),
	)
	username := ""
	backend.SetAuthType(pgproto3.AuthTypeMD5Password)
	switch startupMsg.(type) {
	case *pgproto3.StartupMessage:
		buf := (&pgproto3.AuthenticationMD5Password{
			Salt: types.MD5AuthSalt,
		}).Encode(nil)
		_, err = conn.Write(buf)
		if err != nil {
			return errors.Wrap(err, "sending AuthenticationMD5Password to client")
		}
		msg, err := backend.Receive()
		if err != nil {
			return errors.Wrap(err, "receive message from client")
		}
		msgPass := msg.(*pgproto3.PasswordMessage)
		logger.Info(
			"got password message",
			zap.String("event", "authentication"),
			zap.String("password", msgPass.Password),
		)
		username = startupMsg.(*pgproto3.StartupMessage).Parameters["user"]
		if auth.GetProvider().CheckAuth(username, msgPass.Password) {
			buf = (&pgproto3.AuthenticationOk{}).Encode(nil)
		} else {
			buf = (&pgproto3.ErrorResponse{
				Severity: "ERROR",
				Code:     "28000", // 28P01 - invalid password
				Message:  "password authentication failed for user",
			}).Encode(nil)
		}
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

	logger.Debug(
		"user logged in",
		zap.String("event", "authentication"),
	)
	// Read and handle incoming messages
mainLoop:
	for {
		msg, err := backend.Receive()
		if err != nil {
			return errors.Wrap(err, "receive client query")
		}

		switch msg := msg.(type) {
		case *pgproto3.Query:
			logger.Debug(
				"received query",
				zap.String("event", "running_query"),
				zap.String("query", msg.String),
			)
			accessInfo, err := queryhelper.GetRelatedTables(msg.String)
			if err != nil {
				buf := (&pgproto3.ErrorResponse{
					Message:  err.(*pg_query.Error).Message,
					File:     err.(*pg_query.Error).Filename,
					Detail:   err.(*pg_query.Error).Context,
					Line:     int32(err.(*pg_query.Error).Lineno),
					Position: int32(err.(*pg_query.Error).Cursorpos),
				}).Encode(nil)
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
				continue mainLoop
			}
			isRead := true
			for _, i := range accessInfo {
				switch auth.GetProvider().CheckAccess(i, username) {
				case false:
					buf := (&pgproto3.ErrorResponse{
						Code:       "42501",
						SchemaName: i.Schema,
						TableName:  i.Name,
						Message:    "You don't have access here",
					}).Encode(nil)
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
					continue mainLoop
				}
				if i.AccessMode != types.Select {
					isRead = false
				}
			}
			result, err := connection.RunQuery(msg.String, isRead)
			if err != nil {
				switch e := err.(type) {
				case *pgconn.PgError:
					buf := (&pgproto3.ErrorResponse{
						Severity:         e.Severity,
						Code:             e.Code,
						Message:          e.Message,
						Detail:           e.Detail,
						Hint:             e.Hint,
						Position:         e.Position,
						InternalPosition: e.InternalPosition,
						InternalQuery:    e.InternalQuery,
						Where:            e.Where,
						SchemaName:       e.SchemaName,
						TableName:        e.TableName,
						ColumnName:       e.ColumnName,
						DataTypeName:     e.DataTypeName,
						ConstraintName:   e.ConstraintName,
						File:             e.File,
						Line:             e.Line,
						Routine:          e.Routine,
					}).Encode(nil)
					_, err = conn.Write(buf)
					if err != nil {
						return errors.Wrap(err, "writing query error response")
					}
					buf = (&pgproto3.CommandComplete{
						CommandTag: result.CommandTag,
					}).Encode(nil)
					_, err = conn.Write(buf)
					if err != nil {
						return errors.Wrap(err, "writing CommandComplete response")
					}
					buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)
					_, err = conn.Write(buf)
					if err != nil {
						return errors.Wrap(err, "writing ReadyForQuery response")
					}
					continue mainLoop
				default:
					return errors.Wrap(err, "getting result from postgres")
				}
			}
			if len(result.DataRows) > 0 {
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
			} else {
				buf := (&pgproto3.EmptyQueryResponse{}).Encode(nil)
				_, err = conn.Write(buf)
				if err != nil {
					return errors.Wrap(err, "writing query response")
				}
			}
			buf := (&pgproto3.CommandComplete{
				CommandTag: result.CommandTag,
			}).Encode(nil)
			_, err = conn.Write(buf)
			if err != nil {
				return errors.Wrap(err, "writing query response")
			}
			buf = (&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)
			_, err = conn.Write(buf)
			if err != nil {
				return errors.Wrap(err, "writing query response")
			}
		case *pgproto3.Terminate:
			logger.Info(
				"received terminate message",
				zap.String("event", "termination"),
			)
			return nil
		default:
			return errors.Errorf("received unhandled message: %+v", msg)
		}
	}
}
