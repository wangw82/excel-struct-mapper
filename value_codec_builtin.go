package mapper

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

type builtinValueCodec struct{}

func (builtinValueCodec) Decode(_ context.Context, _ ValueCodecContext, cells []Cell, target reflect.Type) (reflect.Value, error) {
	if len(cells) == 0 {
		cells = []Cell{{}}
	}
	value := reflect.New(target).Elem()
	if err := decodeBuiltin(value, cells); err != nil {
		return reflect.Value{}, err
	}
	return value, nil
}

func (builtinValueCodec) Encode(_ context.Context, _ ValueCodecContext, value reflect.Value) ([]string, error) {
	return encodeBuiltin(value)
}

func decodeBuiltin(destination reflect.Value, cells []Cell) error {
	if destination.Kind() == reflect.Pointer {
		destination.Set(reflect.New(destination.Type().Elem()))
		return decodeBuiltin(destination.Elem(), cells)
	}
	text := cells[0].Value
	if destination.Type() == timeType {
		value, err := ParseDate(text, "", false)
		if err != nil {
			return err
		}
		destination.Set(reflect.ValueOf(value))
		return nil
	}
	if destination.CanAddr() {
		if decoder, ok := destination.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return decoder.UnmarshalText([]byte(text))
		}
	}
	switch destination.Kind() {
	case reflect.String:
		destination.SetString(text)
	case reflect.Bool:
		value, err := strconv.ParseBool(text)
		if err != nil {
			return err
		}
		destination.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(text, 10, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(text, 10, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(text, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetFloat(value)
	case reflect.Slice:
		if destination.Type().Elem().Kind() == reflect.String {
			value := reflect.MakeSlice(destination.Type(), len(cells), len(cells))
			for i := range cells {
				value.Index(i).SetString(cells[i].Value)
			}
			destination.Set(value)
			return nil
		}
		return json.Unmarshal([]byte(text), destination.Addr().Interface())
	case reflect.Map, reflect.Struct:
		return json.Unmarshal([]byte(text), destination.Addr().Interface())
	default:
		return fmt.Errorf("unsupported type %s", destination.Type())
	}
	return nil
}

func encodeBuiltin(value reflect.Value) ([]string, error) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return []string{""}, nil
		}
		value = value.Elem()
	}
	if value.Type() == timeType {
		return []string{value.Interface().(time.Time).Format(time.RFC3339)}, nil
	}
	if value.CanInterface() {
		if encoder, ok := value.Interface().(encoding.TextMarshaler); ok {
			text, err := encoder.MarshalText()
			return []string{string(text)}, err
		}
	}
	switch value.Kind() {
	case reflect.String:
		return []string{value.String()}, nil
	case reflect.Bool:
		return []string{strconv.FormatBool(value.Bool())}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{strconv.FormatInt(value.Int(), 10)}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []string{strconv.FormatUint(value.Uint(), 10)}, nil
	case reflect.Float32, reflect.Float64:
		return []string{strconv.FormatFloat(value.Float(), 'g', -1, value.Type().Bits())}, nil
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.String {
			if value.Len() == 0 {
				return []string{""}, nil
			}
			result := make([]string, value.Len())
			for i := range result {
				result[i] = value.Index(i).String()
			}
			return result, nil
		}
		fallthrough
	case reflect.Map, reflect.Struct:
		data, err := json.Marshal(value.Interface())
		return []string{string(data)}, err
	default:
		return nil, fmt.Errorf("unsupported type %s", value.Type())
	}
}
