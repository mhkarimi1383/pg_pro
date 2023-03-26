package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	pg_query "github.com/pganalyze/pg_query_go/v4/parser"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/mhkarimi1383/pg_pro/auth"
	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/connection"
	"github.com/mhkarimi1383/pg_pro/logger"
	msghelper "github.com/mhkarimi1383/pg_pro/msg_helper"
	queryhelper "github.com/mhkarimi1383/pg_pro/query_helper"
	"github.com/mhkarimi1383/pg_pro/types"
	"github.com/mhkarimi1383/pg_pro/utils"
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
		err := msghelper.WriteMessage(&pgproto3.AuthenticationMD5Password{
			Salt: types.MD5AuthSalt,
		}, conn)
		if err != nil {
			return err
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
			err := msghelper.WriteMessage(&pgproto3.AuthenticationOk{}, conn)
			if err != nil {
				return err
			}

			err = msghelper.WriteMessage(&pgproto3.ParameterStatus{
				Name:  "server_version",
				Value: config.GetString("pg_version"),
			}, conn)
			if err != nil {
				return err
			}
		} else {
			err := msghelper.WriteMessage(&pgproto3.ErrorResponse{
				Severity: "ERROR",
				Code:     "28000", // 28P01 - invalid password
				Message:  "password authentication failed for user",
			}, conn)
			if err != nil {
				return err
			}
		}
		err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
		if err != nil {
			return err
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

	err = msghelper.WriteMessage(&pgproto3.ParameterStatus{
		Name:  "is_superuser",
		Value: strconv.FormatBool(auth.GetProvider().IsSuperUser(username)),
	}, conn)
	if err != nil {
		return err
	}

	// Read and handle incoming messages
mainLoop:
	for {
		msg, err := backend.Receive()
		if err != nil {
			return errors.Wrap(err, "receive client query")
		}

		logger.Info(
			"received message",
			zap.String("type", utils.GetType(msg)),
		)

		switch msg := msg.(type) {
		case *pgproto3.Query:
			accessInfo, err := queryhelper.GetRelatedTables(msg.String)
			if err != nil {
				err = msghelper.WriteMessage(&pgproto3.ErrorResponse{
					Message:  err.(*pg_query.Error).Message,
					File:     err.(*pg_query.Error).Filename,
					Detail:   err.(*pg_query.Error).Context,
					Line:     int32(err.(*pg_query.Error).Lineno),
					Position: int32(err.(*pg_query.Error).Cursorpos),
				}, conn)
				if err != nil {
					return err
				}

				err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
				if err != nil {
					return err
				}

				err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
				if err != nil {
					return err
				}
				continue mainLoop
			}
			isRead := true
			for _, i := range accessInfo {
				switch auth.GetProvider().CheckAccess(i, username) {
				case false:
					err := msghelper.WriteMessage(&pgproto3.ErrorResponse{
						Code:       "42501",
						SchemaName: i.Schema,
						TableName:  i.Name,
						Message:    "You don't have access here",
					}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
					if err != nil {
						return err
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
					err = msghelper.WriteMessage(&pgproto3.ErrorResponse{
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
					}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
					if err != nil {
						return err
					}

					continue mainLoop
				default:
					return errors.Wrap(err, "getting result from postgres")
				}
			}
			if len(result.DataRows) > 0 {
				err = msghelper.WriteMessage(&result.RowDescription, conn)
				if err != nil {
					return err
				}

				for _, d := range result.DataRows {
					err = msghelper.WriteMessage(&d, conn)
					if err != nil {
						return err
					}
				}
			} else {
				err = msghelper.WriteMessage(&pgproto3.EmptyQueryResponse{}, conn)
				if err != nil {
					return err
				}
			}

			err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
			if err != nil {
				return err
			}

			err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
			if err != nil {
				return err
			}

		case *pgproto3.Parse:
			accessInfo, err := queryhelper.GetRelatedTables(msg.Query)
			if err != nil {
				err = msghelper.WriteMessage(&pgproto3.ErrorResponse{
					Message:  err.(*pg_query.Error).Message,
					File:     err.(*pg_query.Error).Filename,
					Detail:   err.(*pg_query.Error).Context,
					Line:     int32(err.(*pg_query.Error).Lineno),
					Position: int32(err.(*pg_query.Error).Cursorpos),
				}, conn)
				if err != nil {
					return err
				}

				err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
				if err != nil {
					return err
				}

				err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
				if err != nil {
					return err
				}
				continue mainLoop
			}
			isRead := true
			for _, i := range accessInfo {
				switch auth.GetProvider().CheckAccess(i, username) {
				case false:
					err := msghelper.WriteMessage(&pgproto3.ErrorResponse{
						Code:       "42501",
						SchemaName: i.Schema,
						TableName:  i.Name,
						Message:    "You don't have access here",
					}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
					if err != nil {
						return err
					}
					continue mainLoop
				}
				if i.AccessMode != types.Select {
					isRead = false
				}
			}
			ii := []any{}
			for _, i := range msg.ParameterOIDs {
				ii = append(ii, strconv.Itoa(int(i)))
			}
			result, err := connection.RunQuery(msg.Query, isRead, ii...)
			log.Println("====================================")
			log.Println(msg.Query, msg.ParameterOIDs)
			log.Println("====================================")

			if err != nil {
				switch e := err.(type) {
				case *pgconn.PgError:
					err = msghelper.WriteMessage(&pgproto3.ErrorResponse{
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
					}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
					if err != nil {
						return err
					}

					err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
					if err != nil {
						return err
					}

					continue mainLoop
				default:
					return errors.Wrap(err, "getting result from postgres")
				}
			}
			if len(result.DataRows) > 0 {
				err = msghelper.WriteMessage(&result.RowDescription, conn)
				if err != nil {
					return err
				}

				for _, d := range result.DataRows {
					err = msghelper.WriteMessage(&d, conn)
					if err != nil {
						return err
					}
				}
			} else {
				err = msghelper.WriteMessage(&pgproto3.EmptyQueryResponse{}, conn)
				if err != nil {
					return err
				}
			}

			err = msghelper.WriteMessage(&pgproto3.CommandComplete{}, conn)
			if err != nil {
				return err
			}

			err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
			if err != nil {
				return err
			}

		case *pgproto3.Terminate:
			logger.Info(
				"received terminate message",
				zap.String("event", "termination"),
			)
			return nil

		case *pgproto3.Sync:
			err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
			if err != nil {
				return err
			}

		case *pgproto3.Describe:
			err = msghelper.WriteMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}, conn)
			if err != nil {
				return err
			}

		// FIXME: not working, I was not able find currect required response from docs.
		case *pgproto3.Execute:
			err = msghelper.WriteMessage(&pgproto3.EmptyQueryResponse{}, conn)
			if err != nil {
				return err
			}

		default:
			logger.Warn(fmt.Sprintf("received unhandled message of type %v: %+v sending to upstream directly\n", utils.GetType(msg), msg))
			fConn, err := connection.GetRawConnection()
			if err != nil {
				return err
			}

			d, err := msghelper.WriteMessageAndRead(msg, fConn)
			if err != nil {
				return err
			}

			log.Println("d", len(d))

			_, err = conn.Write(d)
			if err != nil {
				return err
			}
		}
	}
}
