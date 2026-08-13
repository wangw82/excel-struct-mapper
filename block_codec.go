package mapper

import (
	"context"
	"fmt"
	"reflect"
)

type BlockCodecContext struct {
	Sheet   string
	Block   string
	Options string
}

// BlockCodec converts one selected worksheet block to and from a model field.
// ValueCodec should be used for individual columns inside standard table blocks.
type BlockCodec interface {
	Decode(context.Context, BlockCodecContext, Block, reflect.Type) (reflect.Value, error)
	Encode(context.Context, BlockCodecContext, reflect.Value) ([]Line, error)
}

// MappingValidator runs after a value is decoded and before it is encoded.
// Implement it on a pointer receiver when validation needs the complete value.
type MappingValidator interface {
	ValidateMapping(context.Context) error
}

type BlockCodecFunc struct {
	DecodeFunc func(context.Context, BlockCodecContext, Block, reflect.Type) (reflect.Value, error)
	EncodeFunc func(context.Context, BlockCodecContext, reflect.Value) ([]Line, error)
}

func (f BlockCodecFunc) Decode(ctx context.Context, c BlockCodecContext, block Block, target reflect.Type) (reflect.Value, error) {
	if f.DecodeFunc == nil {
		return reflect.Value{}, fmt.Errorf("block codec does not support decoding")
	}
	return f.DecodeFunc(ctx, c, block, target)
}

func (f BlockCodecFunc) Encode(ctx context.Context, c BlockCodecContext, value reflect.Value) ([]Line, error) {
	if f.EncodeFunc == nil {
		return nil, fmt.Errorf("block codec does not support encoding")
	}
	return f.EncodeFunc(ctx, c, value)
}

// BlockBinding attaches a custom block codec and workflow to one model field.
type BlockBinding struct {
	Key      string
	Field    string
	Workflow BlockWorkflow
	Request  BlockRequest
	Codec    BlockCodec
}
