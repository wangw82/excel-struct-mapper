package mapper

import (
	"context"
	"fmt"
	"reflect"
)

type ValueCodecContext struct {
	Sheet   string
	Block   string
	Field   string
	Cell    Cell
	Options string
}

// ValueCodec converts between worksheet cells and one Go field value.
// It is intentionally separate from BlockCodec: the two extension points work
// at different levels of the mapping pipeline.
type ValueCodec interface {
	Decode(context.Context, ValueCodecContext, []Cell, reflect.Type) (reflect.Value, error)
	Encode(context.Context, ValueCodecContext, reflect.Value) ([]string, error)
}

type ValueCodecFunc struct {
	DecodeFunc func(context.Context, ValueCodecContext, []Cell, reflect.Type) (reflect.Value, error)
	EncodeFunc func(context.Context, ValueCodecContext, reflect.Value) ([]string, error)
}

func (f ValueCodecFunc) Decode(ctx context.Context, c ValueCodecContext, cells []Cell, target reflect.Type) (reflect.Value, error) {
	if f.DecodeFunc == nil {
		return reflect.Value{}, fmt.Errorf("value codec does not support decoding")
	}
	return f.DecodeFunc(ctx, c, cells, target)
}

func (f ValueCodecFunc) Encode(ctx context.Context, c ValueCodecContext, value reflect.Value) ([]string, error) {
	if f.EncodeFunc == nil {
		return nil, fmt.Errorf("value codec does not support encoding")
	}
	return f.EncodeFunc(ctx, c, value)
}
