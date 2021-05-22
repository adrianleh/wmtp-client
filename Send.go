package clientlib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/adrianleh/WTMP-middleend/command"
	"github.com/adrianleh/WTMP-middleend/types"
	"io"
	"reflect"
	"strings"
)

var refKind2TypPrefix = map[reflect.Kind]string{
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

func writeBE(data interface{}, out io.Writer) error {
	return binary.Write(out, binary.BigEndian, data)
}

func serialize(typ types.Type, msg interface{}, out io.Writer) error {
	msgKind := reflect.ValueOf(msg).Kind()
	expTypName := typ.Name()
	actTypName, isValidType := refKind2TypPrefix[msgKind]
	if !isValidType {
		return errors.New("invalid msg type " + msgKind.String())
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
		err := serialize(typ.Typ, arrayVal.Index(i).Interface(), out)
		if err != nil {
			return err
		}
	}
	return nil
}

func Send(typ types.Type, target string, msg interface{}) error {
	typeSer := typ.Serialize()
	msgSize := typ.Size()
	size := uint64(4+4+len(target)+len(typeSer)) + msgSize

	headerBytes, headerErr := makeCommandHeader(command.SendCommandId, size)
	if headerErr != nil {
		return headerErr
	}

	out := bytes.NewBuffer(headerBytes)
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
	return sendViaSocket(out.Bytes())
}
