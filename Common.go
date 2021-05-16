package clientlib

import (
	"encoding/binary"
	"errors"
	"github.com/google/uuid"
	"io"
	"net"
	"sync"
)

const svSockPath = "/tmp/wtmp.sock"

var uuid_ = uuid.Nil
var recvConn net.Conn = nil
var recvConnLock sync.Mutex
var recvConnCond sync.Cond

func sendViaSocket(data *[]byte) error {
	c, sockErr := net.Dial("unix", svSockPath)
	if sockErr != nil {
		return sockErr
	}
	_, sendErr := c.Write(*data)
	return sendErr
}

func writeBE(data interface{}, out io.Writer) error {
	return binary.Write(out, binary.BigEndian, data)
}

func writeCommandHeader(commandCode uint8, size uint64, out io.Writer) error {
	if uuid_ == uuid.Nil {
		return errors.New("attempt to issue command but uuid has not been set")
	}
	if _, err := out.Write(uuid_[:]); err != nil {
		return err
	}
	commandBytes := []byte{commandCode}
	if _, err := out.Write(commandBytes); err != nil {
		return err
	}
	return writeBE(size, out)
}
