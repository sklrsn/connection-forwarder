package guacd

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type ForwarderConnection struct {
	Forward *GuacamoleConnection
	Reverse *GuacamoleConnection

	targetAddr string

	once  sync.Once
	m     sync.RWMutex
	condn sync.Cond
}

func NewForwarderConnection(srcConn, guacdConn net.Conn) (*ForwarderConnection, error) {
	return &ForwarderConnection{
		Reverse: &GuacamoleConnection{
			conn: srcConn,
		},
		Forward: &GuacamoleConnection{
			conn: guacdConn,
			br:   bufio.NewReader(guacdConn),
		},
	}, nil
}

type GuacamoleConnection struct {
	conn net.Conn
	br   *bufio.Reader
}

func (gc *GuacamoleConnection) ReadGuacamoleMessage() (GuacamoleMessage, error) {
	rawMessage, err := gc.readBytes()
	if err != nil {
		return GuacamoleMessage{}, err
	}
	return gc.DeSerializeRawMessage(rawMessage), nil
}

func (gc *GuacamoleConnection) WriteGuacamoleMessage(gcm GuacamoleMessage) error {
	msg, err := gc.SerializeGuacamoleMessage(gcm)
	if err != nil {
		return err
	}

	if _, err := gc.writeBytes([]byte(msg)); err != nil {
		return err
	}

	return nil
}

func (gc *GuacamoleConnection) readBytes() ([]byte, error) {
	return gc.br.ReadBytes(';')
}
func (gc *GuacamoleConnection) writeBytes(msg []byte) (int, error) {
	return gc.conn.Write(msg)
}

type GuacamoleMessage struct {
	OpCode string
	Args   []string
}

func (gc *GuacamoleConnection) SerializeGuacamoleMessage(gm GuacamoleMessage) (string, error) {
	var wireMessage strings.Builder

	args := make([]string, 0)
	args = append(args, fmt.Sprintf("%v.%v", len(gm.OpCode), gm.OpCode))

	if len(gm.Args) > 0 {
		for _, arg := range gm.Args {
			args = append(args, fmt.Sprintf("%v.%v", len(arg), arg))
		}
	} else {
		args = append(args, fmt.Sprintf("%v.", 0))
	}

	if _, err := wireMessage.WriteString(strings.Join(args, ",")); err != nil {
		return "", err
	}

	if _, err := wireMessage.WriteString(";"); err != nil {
		return "", err
	}

	return wireMessage.String(), nil
}

func (gc *GuacamoleConnection) SerializeRawMessage(args []string) (string, error) {
	var msg strings.Builder

	for _, arg := range args {
		if _, err := msg.WriteString(fmt.Sprintf("%v.%v", len(arg), arg)); err != nil {
			return "", err
		}
	}

	if _, err := msg.WriteString(";"); err != nil {
		return "", err
	}

	return msg.String(), nil
}

func (gc *GuacamoleConnection) DeSerializeRawMessage(message []byte) GuacamoleMessage {
	var gm GuacamoleMessage

	msgStr := strings.TrimSpace(strings.ReplaceAll(string(message), ";", ""))

	args := strings.Split(msgStr, ",")
	for idx, msg := range args {
		arg := strings.Split(msg, ".")
		if idx == 0 {
			gm.OpCode = arg[1]
		} else {
			gm.Args = append(gm.Args, arg[1])
		}
	}

	return gm
}
