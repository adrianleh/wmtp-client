package clientlib

import (
	"errors"
	"github.com/adrianleh/WTMP-middleend/command"
	"github.com/adrianleh/WTMP-middleend/types"
)

func AcceptType(typ types.Type) error {
	typSer := typ.Serialize()
	msgSize := uint64(len(typSer))
	msg, headerErr := makeCommandHeader(command.AcceptTypeCommandId, msgSize)
	if headerErr != nil {
		return headerErr
	}
	msg = append(msg, typSer...)

	commLock.Lock()
	defer commLock.Unlock()

	if err := sendViaSocket(msg); err != nil {
		return err
	}

	recvConnLock.Lock()
	for recvConn == nil {
		recvConnCond.Wait()
	}
	recvConnLock.Unlock()

	success := make([]byte, 1)
	if _, err := recvConn.Read(success); err != nil {
		return err
	}
	if success[0] == 0 {
		return nil
	} else {
		return errors.New("middle-end rejected AcceptType command")
	}
}
