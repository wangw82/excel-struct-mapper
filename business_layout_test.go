package mapper

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestOptionalTitleBlockMayBeMissingOrEmpty(t *testing.T) {
	type row struct {
		ID int `excel:"header=ID"`
	}
	type model struct {
		Optional []row `excel:"key=optional;workflow=title;title=OPTIONAL;format=slice;blank_line=true;optional=true;min_rows=2"`
		Required []row `excel:"key=required;workflow=title;title=REQUIRED;format=slice;blank_line=true"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	for _, rows := range [][][]string{
		{{"REQUIRED"}, {"ID"}, {"7"}, {}},
		{{"OPTIONAL"}, {}, {"REQUIRED"}, {"ID"}, {"7"}, {}},
	} {
		var got model
		if err := plan.Decode(context.Background(), NewSheet("Data", rows), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Required, []row{{ID: 7}}) || got.Optional != nil {
			t.Fatalf("got = %#v", got)
		}
	}
	sheet, err := plan.Encode(context.Background(), model{Required: []row{{ID: 7}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(flattenValues(sheet.Values()), "|"), "OPTIONAL") {
		t.Fatalf("optional block was encoded: %#v", sheet.Values())
	}
}

func TestOptionalRangeDoesNotHideMissingEndBoundary(t *testing.T) {
	type row struct {
		ID int `excel:"header=ID"`
	}
	type model struct {
		Rows []row `excel:"key=rows;workflow=title_range;title=BEGIN;end_title=END;format=slice;optional=true"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	var got model
	err = plan.Decode(context.Background(), NewSheet("Data", [][]string{{"BEGIN"}, {"ID"}, {"1"}}), &got)
	if err == nil {
		t.Fatal("missing end boundary was ignored")
	}
}

func TestLabeledSingleRoundTrip(t *testing.T) {
	type model struct {
		Limit int `excel:"key=limit;workflow=index;start_row=1;end_row=1;format=single;label=LIMIT;label_col=1;value_col=2"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	want := model{Limit: 12}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sheet.Values(), [][]string{{"LIMIT", "12"}}) {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got = %#v", got)
	}
	if err := plan.Decode(context.Background(), NewSheet("Data", [][]string{{"OTHER", "12"}}), &got); err == nil {
		t.Fatal("invalid label accepted")
	}
}

func TestSingleUsesRegisteredValueCodec(t *testing.T) {
	registry := NewRegistry()
	codec := ValueCodecFunc{
		DecodeFunc: func(_ context.Context, codecContext ValueCodecContext, cells []Cell, target reflect.Type) (reflect.Value, error) {
			value := reflect.New(target).Elem()
			value.SetString(strings.TrimPrefix(cells[0].Value, codecContext.Options))
			return value, nil
		},
		EncodeFunc: func(_ context.Context, codecContext ValueCodecContext, value reflect.Value) ([]string, error) {
			return []string{codecContext.Options + value.String()}, nil
		},
	}
	if err := registry.RegisterValueCodec("prefixed", codec); err != nil {
		t.Fatal(err)
	}
	type model struct {
		Value string `excel:"key=value;workflow=index;start_row=1;end_row=1;format=single;label=VALUE;value_codec=prefixed;codec_options=v:"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	want := model{Value: "data"}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got = %#v, sheet = %#v", got, sheet.Values())
	}
}

func TestBlockCodecCanOwnInclusiveTitleRange(t *testing.T) {
	type compound struct {
		First  string
		Second string
	}
	codec := BlockCodecFunc{
		DecodeFunc: func(_ context.Context, codecContext BlockCodecContext, block Block, target reflect.Type) (reflect.Value, error) {
			if codecContext.Options != "pair" {
				return reflect.Value{}, errors.New("codec options were not propagated")
			}
			value := reflect.New(target).Elem()
			value.FieldByName("First").SetString(block.Rows[1][0].Value)
			value.FieldByName("Second").SetString(block.Rows[3][0].Value)
			return value, nil
		},
		EncodeFunc: func(_ context.Context, codecContext BlockCodecContext, value reflect.Value) ([]Line, error) {
			if codecContext.Options != "pair" {
				return nil, errors.New("codec options were not propagated")
			}
			return []Line{
				{"BEGIN"}, {value.FieldByName("First").String()},
				{"END"}, {value.FieldByName("Second").String()}, {},
			}, nil
		},
	}
	registry := NewRegistry()
	if err := registry.RegisterBlockCodec("compound", codec); err != nil {
		t.Fatal(err)
	}
	type tailRow struct {
		ID int `excel:"header=ID"`
	}
	type model struct {
		Compound compound  `excel:"key=compound;workflow=title_range;title=BEGIN;end_title=END;format=struct;block_codec=compound;codec_options=pair;include_end_block=true"`
		Tail     []tailRow `excel:"key=tail;workflow=title;title=TAIL;format=slice;blank_line=true"`
	}
	want := model{Compound: compound{First: "a", Second: "b"}, Tail: []tailRow{{ID: 7}}}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, sheet = %#v", got, sheet.Values())
	}
}

func TestFormRoundTrip(t *testing.T) {
	type metadata struct {
		Name   string   `excel:"header=Name;required=true"`
		Limits []string `excel:"header=Limits;multi_cell=true;separator=/"`
	}
	type model struct {
		Metadata metadata `excel:"key=metadata;workflow=title;title=META;format=form;blank_line=true"`
	}
	want := model{Metadata: metadata{Name: "Alpha", Limits: []string{"10", "20"}}}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, sheet = %#v", got, sheet.Values())
	}
}

func TestTransposeRoundTrip(t *testing.T) {
	type entry struct {
		Region string `excel:"header=Region;required=true"`
		Limit  int    `excel:"header=Limit;required=true"`
	}
	type model struct {
		Entries []entry `excel:"key=entries;workflow=title;title=RECORDS;format=transpose;blank_line=true"`
	}
	want := model{Entries: []entry{{Region: "A", Limit: 10}, {Region: "B", Limit: 20}}}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, sheet = %#v", got, sheet.Values())
	}
}

func TestSeparatedMultiCellFieldsRoundTrip(t *testing.T) {
	type row struct {
		ID     int      `excel:"header=ID"`
		First  []string `excel:"header=First;multi_cell=true;separator=/"`
		Second []string `excel:"header=Second;multi_cell=true"`
	}
	type model struct {
		Rows []row `excel:"key=rows;workflow=all;format=slice"`
	}
	want := model{Rows: []row{{ID: 1, First: []string{"a", "b"}, Second: []string{"c", "d"}}}}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, sheet = %#v", got, sheet.Values())
	}
}

func TestRegisteredBlockCodecCanBeSelectedByTag(t *testing.T) {
	codec := BlockCodecFunc{
		DecodeFunc: func(_ context.Context, _ BlockCodecContext, block Block, target reflect.Type) (reflect.Value, error) {
			value := reflect.New(target).Elem()
			value.SetString(strings.TrimPrefix(block.Rows[0][0].Value, "encoded:"))
			return value, nil
		},
		EncodeFunc: func(_ context.Context, _ BlockCodecContext, value reflect.Value) ([]Line, error) {
			return []Line{{"encoded:" + value.String()}}, nil
		},
	}
	registry := NewRegistry()
	if err := registry.RegisterBlockCodec("prefixed_block", codec); err != nil {
		t.Fatal(err)
	}
	type model struct {
		Value string `excel:"key=value;workflow=all;format=single;block_codec=prefixed_block"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	want := model{Value: "data"}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got = %#v", got)
	}
}

func TestNewLayoutCompileFailures(t *testing.T) {
	type row struct {
		Values []string `excel:"header=Values;multi_cell=true"`
	}
	type unknownCodec struct {
		Value string `excel:"key=value;workflow=all;format=single;block_codec=missing"`
	}
	if _, err := Compile[unknownCodec](); !errors.Is(err, ErrUnknownBlockCodec) {
		t.Fatalf("unknown codec error = %v", err)
	}
	type badSeparator struct {
		Rows []struct {
			Value string `excel:"header=Value;separator=/"`
		} `excel:"key=rows;workflow=all;format=slice"`
	}
	if _, err := Compile[badSeparator](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("separator error = %v", err)
	}
	type badTranspose struct {
		Rows []row `excel:"key=rows;workflow=all;format=transpose"`
	}
	if _, err := Compile[badTranspose](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("transpose error = %v", err)
	}
	type misplacedValueCodec struct {
		Rows []struct {
			Value string `excel:"header=Value"`
		} `excel:"key=rows;workflow=all;format=slice;value_codec=builtin"`
	}
	if _, err := Compile[misplacedValueCodec](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("misplaced value codec error = %v", err)
	}
	type child struct {
		Rows []struct {
			Value string `excel:"header=Value"`
		} `excel:"key=rows;workflow=all;format=slice"`
	}
	type badGroupCodec struct {
		Group child `excel:"key=group;workflow=all;format=group;block_codec=custom"`
	}
	registry := NewRegistry()
	codec := BlockCodecFunc{
		DecodeFunc: func(context.Context, BlockCodecContext, Block, reflect.Type) (reflect.Value, error) {
			return reflect.Value{}, nil
		},
		EncodeFunc: func(context.Context, BlockCodecContext, reflect.Value) ([]Line, error) { return nil, nil },
	}
	if err := registry.RegisterBlockCodec("custom", codec); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterBlockCodec("custom", codec); err == nil {
		t.Fatal("duplicate block codec accepted")
	}
	if _, err := Compile[badGroupCodec](WithRegistry(registry)); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("group codec error = %v", err)
	}
}

func TestOpenEndedTitleRanges(t *testing.T) {
	type row struct {
		ID int `excel:"header=ID"`
	}
	t.Run("start_to_end", func(t *testing.T) {
		type model struct {
			Rows []row `excel:"key=rows;workflow=title_range;title=BEGIN;format=slice"`
		}
		plan, err := Compile[model]()
		if err != nil {
			t.Fatal(err)
		}
		var got model
		if err := plan.Decode(context.Background(), NewSheet("Data", [][]string{{"BEGIN"}, {"ID"}, {"1"}}), &got); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("beginning_to_end_title", func(t *testing.T) {
		type model struct {
			Rows []row `excel:"key=rows;workflow=title_range;end_title=END;format=slice"`
		}
		plan, err := Compile[model]()
		if err != nil {
			t.Fatal(err)
		}
		var got model
		if err := plan.Decode(context.Background(), NewSheet("Data", [][]string{{"ID"}, {"1"}, {"END"}}), &got); err != nil {
			t.Fatal(err)
		}
	})
}

type validatedLayoutRow struct {
	Value int `excel:"header=Value"`
}

func (r *validatedLayoutRow) ValidateMapping(context.Context) error {
	if r.Value < 0 {
		return errors.New("value must not be negative")
	}
	return nil
}

func TestMappingAndModelValidationLifecycle(t *testing.T) {
	type model struct {
		Rows []validatedLayoutRow `excel:"key=rows;workflow=all;format=slice"`
	}
	modelValidator := func(_ context.Context, value any) error {
		mapped := value.(*model)
		if len(mapped.Rows) == 0 {
			return errors.New("at least one row is required")
		}
		return nil
	}
	plan, err := Compile[model](WithModelValidation(modelValidator))
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), NewSheet("Data", [][]string{{"Value"}, {"-1"}}), &got); err == nil {
		t.Fatal("mapping validation was not called")
	}
	if _, err := plan.Encode(context.Background(), model{}); err == nil {
		t.Fatal("model validation was not called")
	}
	if _, err := plan.Encode(context.Background(), model{Rows: []validatedLayoutRow{{Value: -1}}}); err == nil {
		t.Fatal("mapping validation was not called during encoding")
	}
}

func flattenValues(rows [][]string) []string {
	var values []string
	for _, row := range rows {
		values = append(values, row...)
	}
	return values
}
