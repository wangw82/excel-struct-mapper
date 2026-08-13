package mapper

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestBlockBindingMixesWithTaggedBlocksAndReceivesContext(t *testing.T) {
	type row struct {
		Name string `excel:"header=Name"`
	}
	type config struct {
		Title string
		Rows  []row `excel:"key=rows;workflow=start;start_row=2;format=slice"`
	}
	type contextKey string
	const key contextKey = "prefix"
	codec := BlockCodecFunc{
		DecodeFunc: func(ctx context.Context, c BlockCodecContext, block Block, target reflect.Type) (reflect.Value, error) {
			if c.Sheet != "Data" || c.Block != "title" {
				t.Fatalf("block codec context = %#v", c)
			}
			value := reflect.New(target).Elem()
			value.SetString(ctx.Value(key).(string) + block.Rows[0][0].Value)
			return value, nil
		},
		EncodeFunc: func(ctx context.Context, c BlockCodecContext, value reflect.Value) ([]Line, error) {
			if c.Block != "title" {
				t.Fatalf("block codec context = %#v", c)
			}
			return []Line{{ctx.Value(key).(string) + value.String()}}, nil
		},
	}
	firstRow := BlockWorkflowFunc{
		SelectFunc: func(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
			block, err := finishSelection(ctx, sheet, 0, 1, request)
			return []Block{block}, err
		},
		PlaceFunc: func(_ context.Context, output BlockOutput, _ BlockRequest, blocks [][]Line) error {
			return output.PlaceLines(0, blocks[0]...)
		},
	}
	plan, err := CompileWithBindings[config]([]BlockBinding{{Key: "title", Field: "Title", Workflow: firstRow, Codec: codec}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), key, "ctx:")
	sheet, err := plan.Encode(ctx, config{Title: "Catalog", Rows: []row{{Name: "Chair"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sheet.Values(), [][]string{{"ctx:Catalog"}, {"Name"}, {"Chair"}}) {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got config
	if err := plan.Decode(ctx, NewSheet("Data", [][]string{{"Catalog"}, {"Name"}, {"Chair"}}), &got); err != nil {
		t.Fatal(err)
	}
	want := config{Title: "ctx:Catalog", Rows: []row{{Name: "Chair"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestBlockBindingCompileFailures(t *testing.T) {
	type config struct {
		Value string
	}
	if _, err := CompileWithBindings[config]([]BlockBinding{{Key: "value", Field: "Value"}}); !errors.Is(err, ErrInvalidBlockBinding) {
		t.Fatalf("error = %v", err)
	}
}

func TestThreeBlockBindingsAnonymousTaggedConfigAndContextValuesRoundTrip(t *testing.T) {
	type row struct {
		Name string `excel:"header=Name"`
	}
	type ManualMetadata struct {
		Title   string
		Version string
	}
	type TaggedConfig struct {
		Rows []row `excel:"key=rows;workflow=start;start_row=4;format=slice"`
	}
	type config struct {
		*ManualMetadata
		Note string
		TaggedConfig
	}
	type contextKey string
	const (
		prefixKey contextKey = "prefix"
		suffixKey contextKey = "suffix"
	)
	codec := BlockCodecFunc{
		DecodeFunc: func(ctx context.Context, _ BlockCodecContext, block Block, target reflect.Type) (reflect.Value, error) {
			text := strings.TrimPrefix(block.Rows[0][0].Value, ctx.Value(prefixKey).(string))
			text = strings.TrimSuffix(text, ctx.Value(suffixKey).(string))
			value := reflect.New(target).Elem()
			value.SetString(text)
			return value, nil
		},
		EncodeFunc: func(ctx context.Context, _ BlockCodecContext, value reflect.Value) ([]Line, error) {
			return []Line{{ctx.Value(prefixKey).(string) + value.String() + ctx.Value(suffixKey).(string)}}, nil
		},
	}
	rowWorkflow := BlockWorkflowFunc{
		SelectFunc: func(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
			block, err := finishSelection(ctx, sheet, request.StartRow, request.EndRow+1, request)
			return []Block{block}, err
		},
		PlaceFunc: func(_ context.Context, output BlockOutput, request BlockRequest, blocks [][]Line) error {
			return output.PlaceLines(request.StartRow, blocks[0]...)
		},
	}
	plan, err := CompileWithBindings[config]([]BlockBinding{
		{Key: "title", Field: "ManualMetadata.Title", Workflow: rowWorkflow, Request: BlockRequest{StartRow: 0, EndRow: 0}, Codec: codec},
		{Key: "version", Field: "ManualMetadata.Version", Workflow: rowWorkflow, Request: BlockRequest{StartRow: 1, EndRow: 1}, Codec: codec},
		{Key: "note", Field: "Note", Workflow: rowWorkflow, Request: BlockRequest{StartRow: 2, EndRow: 2}, Codec: codec},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), prefixKey, "[")
	ctx = context.WithValue(ctx, suffixKey, "]")
	want := config{ManualMetadata: &ManualMetadata{Title: "Catalog", Version: "v1"}, TaggedConfig: TaggedConfig{Rows: []row{{Name: "Chair"}}}, Note: "Ready"}
	sheet, err := plan.Encode(ctx, want)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sheet.Values(), [][]string{{"[Catalog]"}, {"[v1]"}, {"[Ready]"}, {"Name"}, {"Chair"}}) {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got config
	if err := plan.Decode(ctx, sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}
