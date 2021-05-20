package clientlib

import (
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
	return sendViaSocket(msg)
}
