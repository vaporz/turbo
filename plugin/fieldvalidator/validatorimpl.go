package fieldvalidator

import "reflect"
import (
	"github.com/golang/protobuf/proto"
	protobuf "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"errors"
	"strconv"
)

func FindOptions(kind reflect.Kind) []*proto.ExtensionDesc {
	result := []*proto.ExtensionDesc{}
	optionMap := proto.RegisteredExtensions(&protobuf.FieldOptions{})
	for _, op := range optionMap {
		switch kind {
		case reflect.String:
			if op.Field >= 50000 && op.Field <= 50199 {
				result = append(result, op)
			}
		default:
		}
	}
	return result
}

func DoValidate(v string, optionType *proto.ExtensionDesc, optionValue interface{}) (returnValue string, err error) {
	switch optionType.Field {
	case E_MaxLength.Field:
		expectLength := *optionValue.(*int32)
		actual := int32(len(v))
		if actual > expectLength {
			return v, errors.New("validate fail! MaxLength, expected:<=" +
				strconv.FormatInt(int64(expectLength), 10) + ", actual:" +
				strconv.FormatInt(int64(actual), 10))
		}
		return v, nil
	case E_Default.Field:
		defaultValue := *optionValue.(*string)
		if len(v) == 0 {
			v = defaultValue
		}
		return v, nil
	case E_NotBlank.Field:
		notBlank := *optionValue.(*bool)
		if notBlank && len(v) == 0 {
			return v, errors.New("validate fail! NotBlank, value is blank")
		}
		return v, nil
	default:
		return "", errors.New("unknown option type")
	}
}
