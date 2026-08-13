package mapper

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type architectureRow struct {
	Name string `excel:"header=Name;required=true"`
}

func TestPlanOwnsResolvedValueCodecs(t *testing.T) {
	registry := NewRegistry()
	codec := ValueCodecFunc{
		DecodeFunc: func(_ context.Context, _ ValueCodecContext, cells []Cell, target reflect.Type) (reflect.Value, error) {
			value := reflect.New(target).Elem()
			value.SetString(strings.TrimPrefix(cells[0].Value, "x:"))
			return value, nil
		},
		EncodeFunc: func(_ context.Context, _ ValueCodecContext, value reflect.Value) ([]string, error) {
			return []string{"x:" + value.String()}, nil
		},
	}
	if err := registry.RegisterValueCodec("prefixed", codec); err != nil {
		t.Fatal(err)
	}
	type row struct {
		Name string `excel:"header=Name;value_codec=prefixed"`
	}
	type model struct {
		Rows []row `excel:"key=rows;workflow=all;format=slice"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}

	// Execution uses only the immutable plan, not the compiler's registry.
	sheet, err := plan.Encode(context.Background(), model{Rows: []row{{Name: "Ada"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if got.Rows[0].Name != "Ada" {
		t.Fatalf("got = %#v", got)
	}
}

func TestPointerBlocksRoundTrip(t *testing.T) {
	type model struct {
		Primary *architectureRow   `excel:"key=primary;workflow=index;start_row=1;end_row=2;format=struct"`
		Others  []*architectureRow `excel:"key=others;workflow=start;start_row=3;format=slice"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	want := model{
		Primary: &architectureRow{Name: "Ada"},
		Others:  []*architectureRow{{Name: "Grace"}, {Name: "Linus"}},
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
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestMultiPointerBlocksRoundTrip(t *testing.T) {
	type model struct {
		Primary **architectureRow   `excel:"key=primary;workflow=index;start_row=1;end_row=2;format=struct"`
		Others  []**architectureRow `excel:"key=others;workflow=start;start_row=3;format=slice"`
	}
	primary := &architectureRow{Name: "Ada"}
	other := &architectureRow{Name: "Grace"}
	want := model{Primary: &primary, Others: []**architectureRow{&other}}
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
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestTitleRangeWorkflowRoundTrip(t *testing.T) {
	type model struct {
		Rows []architectureRow `excel:"key=rows;workflow=title_range;title=BEGIN;end_title=END;format=slice"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	want := model{Rows: []architectureRow{{Name: "Ada"}}}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sheet.Values(), [][]string{{"BEGIN"}, {"Name"}, {"Ada"}, {"END"}}) {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v", got)
	}
}

func TestWorkflowContractAndLayoutAreValidated(t *testing.T) {
	bad := BlockWorkflowFunc{
		SelectFunc: func(context.Context, Sheet, BlockRequest) ([]Block, error) {
			return []Block{{StartRow: -1, EndRow: 99}}, nil
		},
		PlaceFunc: func(_ context.Context, output BlockOutput, _ BlockRequest, blocks [][]Line) error {
			return output.PlaceLines(0, blocks[0]...)
		},
	}
	registry := NewRegistry()
	if err := registry.RegisterBlockWorkflow("bad", bad); err != nil {
		t.Fatal(err)
	}
	type invalidModel struct {
		Rows []architectureRow `excel:"key=rows;workflow=bad;format=slice"`
	}
	plan, err := Compile[invalidModel](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	var got invalidModel
	err = plan.Decode(context.Background(), NewSheet("Data", [][]string{{"Name"}}), &got)
	if !errors.Is(err, ErrInvalidBlock) {
		t.Fatalf("error = %v", err)
	}

	type overlapModel struct {
		A []architectureRow `excel:"key=a;workflow=index;start_row=1;end_row=2;format=slice"`
		B []architectureRow `excel:"key=b;workflow=index;start_row=1;end_row=2;format=slice"`
	}
	overlap, err := Compile[overlapModel]()
	if err != nil {
		t.Fatal(err)
	}
	_, err = overlap.Encode(context.Background(), overlapModel{})
	if !errors.Is(err, ErrLayoutConflict) {
		t.Fatalf("error = %v", err)
	}
}

func TestWorkflowCannotExposeConsumedDelimiterAsContent(t *testing.T) {
	registry := NewRegistry()
	workflow := BlockWorkflowFunc{
		SelectFunc: func(_ context.Context, sheet Sheet, _ BlockRequest) ([]Block, error) {
			return []Block{{
				Rows:           Rows{sheet.Rows[1]},
				StartRow:       0,
				EndRow:         1,
				ConsumedEndRow: 2,
			}}, nil
		},
		PlaceFunc: func(context.Context, BlockOutput, BlockRequest, [][]Line) error { return nil },
	}
	if err := registry.RegisterBlockWorkflow("delimiter_content", workflow); err != nil {
		t.Fatal(err)
	}
	type model struct {
		Rows []architectureRow `excel:"key=rows;workflow=delimiter_content;format=slice"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	var got model
	err = plan.Decode(context.Background(), NewSheet("Data", [][]string{{"Name"}, {}}), &got)
	if !errors.Is(err, ErrInvalidBlock) {
		t.Fatalf("error = %v", err)
	}
}

func TestEncodeRejectsOutputThatViolatesMinimumRows(t *testing.T) {
	type model struct {
		Rows []architectureRow `excel:"key=rows;workflow=all;format=slice;min_rows=3"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	_, err = plan.Encode(context.Background(), model{})
	if !errors.Is(err, ErrLayoutConflict) {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileRejectsAmbiguousOrUnsafeModels(t *testing.T) {
	type wrongCodec struct {
		Rows []architectureRow `excel:"key=rows;workflow=all;format=struct"`
	}
	if _, err := Compile[wrongCodec](); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("error = %v", err)
	}
	type unsupportedOption struct {
		Rows []architectureRow `excel:"key=rows;workflow=all;start_row=2;format=slice"`
	}
	if _, err := Compile[unsupportedOption](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("error = %v", err)
	}
	type privateRow struct {
		name string `excel:"header=Name"`
	}
	type privateModel struct {
		Rows []privateRow `excel:"key=rows;workflow=all;format=slice"`
	}
	if _, err := Compile[privateModel](); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("error = %v", err)
	}
}

func TestDefaultHeaderNormalizationAndSafeFunctionAdapters(t *testing.T) {
	type model struct {
		Rows []architectureRow `excel:"key=rows;workflow=all;format=slice"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	var got model
	if err := plan.Decode(context.Background(), NewSheet("Data", [][]string{{" name "}, {"Ada"}}), &got); err != nil {
		t.Fatal(err)
	}
	if got.Rows[0].Name != "Ada" {
		t.Fatalf("got = %#v", got)
	}
	if _, err := (ValueCodecFunc{}).Encode(context.Background(), ValueCodecContext{}, reflect.Value{}); err == nil {
		t.Fatal("missing encoder accepted")
	}
	if _, err := (BlockCodecFunc{}).Decode(context.Background(), BlockCodecContext{}, Block{}, reflect.TypeOf("")); err == nil {
		t.Fatal("missing decoder accepted")
	}
}
