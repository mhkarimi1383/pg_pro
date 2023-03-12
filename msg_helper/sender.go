package msghelper

import (
	"fmt"
	"net"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/pkg/errors"

	"github.com/mhkarimi1383/pg_pro/logger"
	"github.com/mhkarimi1383/pg_pro/utils"
)

func WriteMessage(msg pgproto3.Message, conn net.Conn) error {
	buf := msg.Encode(nil)
	_, err := conn.Write(buf)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("sending message of type `%v` to the client", utils.GetType(msg)))
	}
	logger.Info(fmt.Sprintf("sent message of type `%v` to `%v`", utils.GetType(msg), conn.RemoteAddr().String()))
	return nil
}

func WriteMessageAndRead(msg pgproto3.Message, conn net.Conn) ([]byte, error) {
	buf := msg.Encode(nil)
	_, err := conn.Write(buf)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("sending message of type `%v` to the client", utils.GetType(msg)))
	}
	logger.Info(fmt.Sprintf("sent message of type `%v` to `%v`", utils.GetType(msg), conn.RemoteAddr().String()))
	var d []byte
	_, err = conn.Read(d)
	return d, err
}
