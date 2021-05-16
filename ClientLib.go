package clientlib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/adrianleh/WTMP-middleend/command"
	"github.com/adrianleh/WTMP-middleend/types"
	"github.com/google/uuid"
	"io"
	"net"
	"reflect"
	"strings"
)

const svSockPath = "/tmp/wtmp.sock"

var uuid_ = uuid.Nil

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

var type2Type = map[reflect.Kind]string{
	reflect.Uint16:  "Char",
	reflect.Int32:   "Int32",
	reflect.Int64:   "Int64",
	reflect.Float32: "Float32",
	reflect.Float64: "Float64",
	reflect.Bool:    "Bool",
	reflect.Struct:  "Struct",
	reflect.Array:   "Array",
	reflect.Slice:   "Array",
}

func checkValidUnion(unionType types.UnionType) error {
	foundStruct := false
	foundArray := false
	for _, memTyp := range unionType.Members {
		if strings.HasPrefix(memTyp.Name(), "Struct") {
			if foundStruct {
				return errors.New("union has too many structs")
			}
			foundStruct = true
		} else if strings.HasPrefix(memTyp.Name(), "Array") {
			if foundArray {
				return errors.New("union has too many arrays")
			}
			foundArray = true
		} else if strings.HasPrefix(memTyp.Name(), "Union") {
			return errors.New("union contains direct union member")
		}
	}
	return nil
}

func castUnion(unionTyp types.UnionType, actTypName string) (types.Type, uint64, error) {
	for _, memTyp := range unionTyp.Members {
		if strings.HasPrefix(memTyp.Name(), actTypName) {
			return memTyp, unionTyp.Size() - memTyp.Size(), nil
		}
	}
	return nil, 0, errors.New(fmt.Sprintf("union %s doesn't contain type %s", unionTyp.Name(), actTypName))
}

func serialize(typ types.Type, msg interface{}, out io.Writer) error {
	msgKind := reflect.ValueOf(msg).Kind()
	expTypName := typ.Name()
	actTypName, isValidType := type2Type[msgKind]
	if !isValidType {
		return errors.New("invalid msg type " + msgKind.String())
	}
	if strings.HasPrefix(expTypName, "Union") {
		unionTyp := typ.(types.UnionType)
		if unionTypErr := checkValidUnion(unionTyp); unionTypErr != nil {
			return unionTypErr
		}
		preciseTyp, numPadding, castUnionErr := castUnion(unionTyp, actTypName)
		if castUnionErr != nil {
			return castUnionErr
		}
		if serErr := serialize(preciseTyp, msg, out); serErr != nil {
			return serErr
		}
		zeroes := make([]byte, numPadding)
		_, writeErr := out.Write(zeroes)
		return writeErr
	}
	if !strings.HasPrefix(expTypName, actTypName) {
		return errors.New(
			fmt.Sprintf("serialization expected type %s but was actually %s", expTypName, actTypName))
	}
	switch msgKind {
	case reflect.Struct:
		return serializeStruct(typ.(types.StructType), msg, out)
	case reflect.Array, reflect.Slice:
		return serializeArray(typ.(types.ArrayType), msg, out)
	default:
		return writeBE(msg, out)
	}
}

func serializeStruct(typ types.StructType, struct_ interface{}, out io.Writer) error {
	structVal := reflect.ValueOf(struct_)
	numFields := structVal.NumField()
	numTypFields := len(typ.Fields)
	if numTypFields != numFields {
		return errors.New(fmt.Sprintf(
			"struct type %s expects %d fields but got %d", typ.Name(), numTypFields, numFields))
	}
	for i := 0; i < numTypFields; i++ {
		err := serialize(typ.Fields[i], structVal.Field(i).Interface(), out)
		if err != nil {
			return err
		}
	}
	return nil
}

func serializeArray(typ types.ArrayType, array_ interface{}, out io.Writer) error {
	arrayVal := reflect.ValueOf(array_)
	len_ := arrayVal.Len()
	if typ.Length != uint64(len_) {
		return errors.New(fmt.Sprintf("array type expects %d elems but got %d", typ.Length, len_))
	}
	for i := 0; i < len_; i++ {
		// TODO this may be unperformant, we may need to use unsafe
		err := serialize(typ.Typ, arrayVal.Index(i), out)
		if err != nil {
			return err
		}
	}
	return nil
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

// Send NOTE: The client does not permit a union over multiple structs or multiple arrays -
//       i.e. it allows at most one array type and at most one struct type per union.
//       Without this simplification, type checking gets really complicated and may
//       have poor performance.
//       Also, a member of a union may not itself be a union.
func Send(typ types.Type, target string, msg interface{}) error {
	typeSer := typ.Serialize()
	msgSize := typ.Size()
	size := uint64(4+4+len(target)+len(typeSer)) + msgSize
	bufInit := make([]byte, 0, 25+size)
	out := bytes.NewBuffer(bufInit)
	if err := writeCommandHeader(command.SendCommandId, size, out); err != nil {
		return err
	}
	if err := writeBE(uint32(len(target)), out); err != nil {
		return err
	}
	if err := writeBE(uint32(len(typeSer)), out); err != nil {
		return err
	}
	if _, err := out.WriteString(target); err != nil {
		return err
	}
	if _, err := out.Write(typeSer); err != nil {
		return err
	}
	if err := serialize(typ, msg, out); err != nil {
		return err
	}
	outBytes := out.Bytes()
	return sendViaSocket(&outBytes)
}
