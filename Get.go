package clientlib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/adrianleh/WTMP-middleend/command"
	"github.com/adrianleh/WTMP-middleend/types"
	"io"
	"reflect"
	"strings"
)

var typStr2RefTyp = map[string]reflect.Type{
	"Char":    reflect.TypeOf(uint16(0)),
	"Int32":   reflect.TypeOf(int32(0)),
	"Int64":   reflect.TypeOf(int64(0)),
	"Float32": reflect.TypeOf(float32(0)),
	"Float64": reflect.TypeOf(float64(0)),
	"Bool":    reflect.TypeOf(false),
}

func typ2RefTyp(typ types.Type) reflect.Type {
	typName := typ.Name()
	refTyp, foundTyp := typStr2RefTyp[typName]
	if foundTyp {
		return refTyp
	}
	if strings.HasPrefix(typName, "Struct") {
		return structTyp2RefTyp(typ.(types.StructType))
	} else if strings.HasPrefix(typName, "Array") {
		return arrayTyp2RefTyp(typ.(types.ArrayType))
	} else {
		panic("impossible type " + typName)
	}
}

func structTyp2RefTyp(typ types.StructType) reflect.Type {
	typFields := typ.Fields
	strFields := make([]reflect.StructField, len(typFields))
	for i, typField := range typFields {
		strFields[i] = reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: typ2RefTyp(typField),
		}
	}
	return reflect.StructOf(strFields)
}

func arrayTyp2RefTyp(typ types.ArrayType) reflect.Type {
	return reflect.ArrayOf(int(typ.Length), typ2RefTyp(typ.Typ))
}

func deserialize(typ types.Type, in io.Reader) (interface{}, error) {
	typName := typ.Name()
	refTyp, foundTyp := typStr2RefTyp[typName]
	if foundTyp {
		ptr := reflect.New(refTyp)
		if err := binary.Read(in, binary.BigEndian, ptr.Interface()); err != nil {
			return nil, err
		}
		return ptr.Elem().Interface(), nil
	}
	if strings.HasPrefix(typName, "Struct") {
		return deserializeStruct(typ.(types.StructType), in)
	} else if strings.HasPrefix(typName, "Array") {
		return deserializeArray(typ.(types.ArrayType), in)
	} else {
		return nil, errors.New("impossible type " + typName)
	}
}

func deserializeStruct(typ types.StructType, in io.Reader) (interface{}, error) {
	refTyp := structTyp2RefTyp(typ)
	strPtr := reflect.New(refTyp)
	for i, typField := range typ.Fields {
		field, err := deserialize(typField, in)
		if err != nil {
			return nil, err
		}
		strPtr.Elem().Field(i).Set(reflect.ValueOf(field))
	}
	return strPtr.Elem().Interface(), nil
}

// TODO may need to improve efficiency by using unsafe or by using pointers to avoid copying
func deserializeArray(typ types.ArrayType, in io.Reader) (interface{}, error) {
	refTyp := arrayTyp2RefTyp(typ)
	arrPtr := reflect.New(refTyp)
	for i := 0; i < int(typ.Length); i++ {
		elem, err := deserialize(typ.Typ, in)
		if err != nil {
			return nil, err
		}
		arrPtr.Elem().Index(i).Set(reflect.ValueOf(elem))
	}
	return arrPtr.Elem().Interface(), nil
}

func Get(typ types.Type) (interface{}, error) {
	typSer := typ.Serialize()
	msgSize := uint64(len(typSer))
	msg, headerErr := makeCommandHeader(command.GetCommandId, msgSize)
	if headerErr != nil {
		return nil, headerErr
	}
	msg = append(msg, typSer...)

	commLock.Lock()
	defer commLock.Unlock()

	if err := sendViaSocket(msg); err != nil {
		return nil, err
	}

	recvConnLock.Lock()
	for recvConn == nil {
		recvConnCond.Wait()
	}
	recvConnLock.Unlock()

	// TODO this isn't super safe - if we read too many or too few bytes,
	//      then future gets will be all scrambled up.
	//      We should probably check for this and panic if anything is untoward.
	return deserialize(typ, recvConn)
}
