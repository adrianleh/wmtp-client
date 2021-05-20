package clientlib

import (
	"encoding/binary"
	"github.com/adrianleh/WTMP-middleend/command"
	"github.com/adrianleh/WTMP-middleend/types"
)

func Empty(typ types.Type) (bool, error) {
	typSer := typ.Serialize()
	msgSize := uint64(len(typSer))
	msg, headerErr := makeCommandHeader(command.EmptyCommandId, msgSize)
	if headerErr != nil {
		return false, headerErr
	}
	msg = append(msg, typSer...)

	commLock.Lock()
	defer commLock.Unlock()

	if err := sendViaSocket(msg); err != nil {
		return false, err
	}

	recvConnLock.Lock()
	for recvConn == nil {
		recvConnCond.Wait()
	}
	recvConnLock.Unlock()

	var isEmpty bool
	if err := binary.Read(recvConn, binary.BigEndian, &isEmpty); err != nil {
		return false, err
	}
	return isEmpty, nil
}
