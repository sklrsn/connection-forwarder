package guacd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type ForwarderConnection struct {
	Forward *GuacamoleConnection
	Reverse *GuacamoleConnection

	once  sync.Once
	m     sync.RWMutex
	condn sync.Cond
}

func NewForwarderConnection(srcConn, guacdConn net.Conn) (*ForwarderConnection, error) {
	return &ForwarderConnection{
		Reverse: &GuacamoleConnection{
			conn:      srcConn,
			br:        bufio.NewReader(srcConn),
			direction: "reverse",
		},
		Forward: &GuacamoleConnection{
			conn:      guacdConn,
			br:        bufio.NewReader(guacdConn),
			direction: "forward",
		},
	}, nil
}

type GuacamoleConnection struct {
	conn      net.Conn
	br        *bufio.Reader
	direction string
}

func (gc *GuacamoleConnection) GetRawConn() net.Conn {
	return gc.conn
}

func (gc *GuacamoleConnection) ReadGuacamoleMessage() (GuacamoleMessage, error) {
	rawMessage, err := gc.readBytes()
	if err != nil {
		return GuacamoleMessage{}, err
	}
	log.Printf("direction=%v reading %v", gc.direction, string(rawMessage))
	return gc.deSerializeRawMessage(rawMessage), nil
}

func (gc *GuacamoleConnection) WriteGuacamoleMessage(gcm GuacamoleMessage) error {
	msg, err := gc.serializeGuacamoleMessage(gcm)
	if err != nil {
		return err
	}

	log.Printf("direction=%v writing %v", gc.direction, string(msg))

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

func (gc *GuacamoleConnection) serializeGuacamoleMessage(gm GuacamoleMessage) (string, error) {
	var wireMessage strings.Builder

	args := make([]string, 0)
	args = append(args, fmt.Sprintf("%v.%v", len(gm.OpCode), gm.OpCode))

	if gm.Args != nil {
		if len(gm.Args) > 0 {
			for _, arg := range gm.Args {
				arg = strings.TrimSpace(arg)
				args = append(args, fmt.Sprintf("%v.%v", len(arg), arg))
			}
		} else {
			args = append(args, fmt.Sprintf("%v.", 0))
		}

		if _, err := wireMessage.WriteString(strings.Join(args, ",")); err != nil {
			return "", err
		}
	} else {
		if _, err := wireMessage.WriteString(string(strings.Join(args, ""))); err != nil {
			return "", err
		}
	}

	if _, err := wireMessage.WriteString(";"); err != nil {
		return "", err
	}

	return wireMessage.String(), nil
}

func (gc *GuacamoleConnection) serializeRawMessage(args []string) (string, error) {
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

func (gc *GuacamoleConnection) deSerializeRawMessage(message []byte) GuacamoleMessage {
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
