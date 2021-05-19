package clientlib

import (
	"encoding/binary"
	"errors"
	"github.com/google/uuid"
	"net"
	"sync"
)

const svSockPath = "/tmp/wtmp.sock"

var uuid_ = uuid.Nil
var recvConn net.Conn = nil
var recvConnLock sync.Mutex
var recvConnCond sync.Cond

func sendViaSocket(data []byte) error {
	c, sockErr := net.Dial("unix", svSockPath)
	if sockErr != nil {
		return sockErr
	}
	_, sendErr := c.Write(data)
	return sendErr
}

func makeCommandHeader(commandCode uint8, size uint64) ([]byte, error) {
	if uuid_ == uuid.Nil {
		return []byte{}, errors.New("attempt to issue command but uuid has not been set")
	}

	ret := make([]byte, 0, 25+size)
	ret = append(ret, uuid_[:]...)
	ret = append(ret, commandCode)
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, size)
	return append(ret, sizeBytes...), nil
}
