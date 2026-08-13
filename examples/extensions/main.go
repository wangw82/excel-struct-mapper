package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	mapper "github.com/wangw82/excel-struct-mapper"
)

type codecRow struct {
	Code string `excel:"header=Code;value_codec=prefixed;codec_options=code:"`
}

type codecModel struct {
	Rows []codecRow `excel:"key=rows;workflow=custom_all;format=slice"`
}

type compoundValue struct {
	First  string
	Second string
}

type blockCodecModel struct {
	Value compoundValue `excel:"key=value;workflow=title_range;title=BEGIN;end_title=END;format=struct;block_codec=compound;codec_options=pair;include_end_block=true"`
}

type tagFields struct {
	Title string `json:"title" ui:"text=Heading,Heading"`
}

type tagModel struct {
	Rows   []codecRow `excel:"key=rows;workflow=all;format=slice"`
	Fields tagFields  `ui:"group=Panel"`
}

type bindingRow struct {
	Name string `excel:"header=Name"`
}

type bindingModel struct {
	Title string
	Rows  []bindingRow `excel:"key=rows;workflow=start;start_row=2;format=slice"`
}

func main() {
	ctx := context.Background()
	registry := mapper.NewRegistry()
	must(registry.RegisterValueCodec("prefixed", prefixedValueCodec()))
	must(registry.RegisterBlockCodec("compound", compoundBlockCodec()))
	must(registry.RegisterBlockWorkflow("custom_all", allRowsWorkflow()))
	must(registry.RegisterTagHandler("ui", "text", mapper.TagHandlerFunc(textTag)))
	must(registry.RegisterTagHandler("ui", "group", mapper.TagHandlerFunc(groupTag)))

	codecPlan := mustCompile[codecModel](mapper.WithRegistry(registry))
	mustRoundTrip(ctx, codecPlan, codecModel{Rows: []codecRow{{Code: "A-7"}}})

	blockPlan := mustCompile[blockCodecModel](mapper.WithRegistry(registry))
	mustRoundTrip(ctx, blockPlan, blockCodecModel{Value: compoundValue{First: "ready", Second: "stable"}})

	tagPlan := mustCompile[tagModel](mapper.WithRegistry(registry))
	results, err := tagPlan.RunTag(ctx, "ui")
	must(err)
	if len(results) != 1 || results[0].Output != "Panel has 1 child" {
		panic(fmt.Sprintf("unexpected tag results: %#v", results))
	}

	bindingPlan, err := mapper.CompileWithBindings[bindingModel]([]mapper.BlockBinding{{
		Key: "title", Field: "Title", Workflow: firstRowWorkflow(), Codec: plainBlockCodec(),
	}})
	must(err)
	mustRoundTrip(ctx, bindingPlan, bindingModel{Title: "Catalog", Rows: []bindingRow{{Name: "Alpha"}}})

	fmt.Println("extensions: value codec, block codec, workflow, application tag, explicit binding")
}

func prefixedValueCodec() mapper.ValueCodec {
	return mapper.ValueCodecFunc{
		DecodeFunc: func(_ context.Context, codecContext mapper.ValueCodecContext, cells []mapper.Cell, target reflect.Type) (reflect.Value, error) {
			value := reflect.New(target).Elem()
			value.SetString(strings.TrimPrefix(cells[0].Value, codecContext.Options))
			return value, nil
		},
		EncodeFunc: func(_ context.Context, codecContext mapper.ValueCodecContext, value reflect.Value) ([]string, error) {
			return []string{codecContext.Options + value.String()}, nil
		},
	}
}

func compoundBlockCodec() mapper.BlockCodec {
	return mapper.BlockCodecFunc{
		DecodeFunc: func(_ context.Context, codecContext mapper.BlockCodecContext, block mapper.Block, target reflect.Type) (reflect.Value, error) {
			if codecContext.Options != "pair" || len(block.Rows) < 4 {
				return reflect.Value{}, fmt.Errorf("invalid compound block")
			}
			value := reflect.New(target).Elem()
			value.FieldByName("First").SetString(block.Rows[1][0].Value)
			value.FieldByName("Second").SetString(block.Rows[3][0].Value)
			return value, nil
		},
		EncodeFunc: func(_ context.Context, codecContext mapper.BlockCodecContext, value reflect.Value) ([]mapper.Line, error) {
			if codecContext.Options != "pair" {
				return nil, fmt.Errorf("invalid codec options")
			}
			return []mapper.Line{
				{"BEGIN"}, {value.FieldByName("First").String()},
				{"END"}, {value.FieldByName("Second").String()}, {},
			}, nil
		},
	}
}

func plainBlockCodec() mapper.BlockCodec {
	return mapper.BlockCodecFunc{
		DecodeFunc: func(_ context.Context, _ mapper.BlockCodecContext, block mapper.Block, target reflect.Type) (reflect.Value, error) {
			value := reflect.New(target).Elem()
			value.SetString(block.Rows[0][0].Value)
			return value, nil
		},
		EncodeFunc: func(_ context.Context, _ mapper.BlockCodecContext, value reflect.Value) ([]mapper.Line, error) {
			return []mapper.Line{{value.String()}}, nil
		},
	}
}

func allRowsWorkflow() mapper.BlockWorkflow {
	return mapper.BlockWorkflowFunc{
		SelectFunc: func(ctx context.Context, sheet mapper.Sheet, _ mapper.BlockRequest) ([]mapper.Block, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return []mapper.Block{{Rows: sheet.Rows, StartRow: 0, EndRow: len(sheet.Rows)}}, nil
		},
		PlaceFunc: func(_ context.Context, output mapper.BlockOutput, _ mapper.BlockRequest, blocks [][]mapper.Line) error {
			return output.PlaceLines(0, blocks[0]...)
		},
	}
}

func firstRowWorkflow() mapper.BlockWorkflow {
	return mapper.BlockWorkflowFunc{
		SelectFunc: func(ctx context.Context, sheet mapper.Sheet, _ mapper.BlockRequest) ([]mapper.Block, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if len(sheet.Rows) == 0 {
				return nil, mapper.ErrSheetEmpty
			}
			return []mapper.Block{{Rows: sheet.Rows[:1], StartRow: 0, EndRow: 1}}, nil
		},
		PlaceFunc: func(_ context.Context, output mapper.BlockOutput, _ mapper.BlockRequest, blocks [][]mapper.Line) error {
			return output.PlaceLines(0, blocks[0]...)
		},
	}
}

func textTag(_ context.Context, tag mapper.TagContext) (any, error) {
	return map[string]string{"field": tag.Field.Name, "label": tag.Params}, nil
}

func groupTag(_ context.Context, tag mapper.TagContext) (any, error) {
	return fmt.Sprintf("%s has %d child", tag.Params, len(tag.Children)), nil
}

func mustRoundTrip[T any](ctx context.Context, plan *mapper.Plan, want T) {
	sheet, err := plan.Encode(ctx, want)
	must(err)
	var got T
	must(plan.Decode(ctx, sheet, &got))
	if !reflect.DeepEqual(got, want) {
		panic(fmt.Sprintf("round trip mismatch: got %#v, want %#v", got, want))
	}
}

func mustCompile[T any](options ...mapper.CompileOption) *mapper.Plan {
	plan, err := mapper.Compile[T](options...)
	must(err)
	return plan
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
