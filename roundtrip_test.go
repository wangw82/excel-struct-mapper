package mapper

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type roundProduct struct {
	ID   int      `excel:"header=ID;required=true"`
	Name string   `excel:"header=Name;allow_empty=true"`
	Tags []string `excel:"header=Tags;multi_cell=true;allow_empty=true"`
}

type roundCatalog struct {
	Products []roundProduct `excel:"key=products;workflow=title;title=Products;format=slice"`
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	plan, err := Compile[roundCatalog]()
	if err != nil {
		t.Fatal(err)
	}
	want := roundCatalog{Products: []roundProduct{{ID: 1, Name: "Chair", Tags: []string{"home", "wood"}}, {ID: 2, Name: "Desk", Tags: []string{"work"}}, {ID: 3, Name: "Empty"}}}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheet.Rows[1]) != 4 || sheet.Rows[3][3].Value != "" {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got roundCatalog
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestBuiltinScalarTypesAndPackageEntries(t *testing.T) {
	type row struct {
		Flag     bool              `excel:"header=Flag"`
		Signed   int8              `excel:"header=Signed"`
		Unsigned uint16            `excel:"header=Unsigned"`
		Ratio    float32           `excel:"header=Ratio"`
		Labels   map[string]string `excel:"header=Labels"`
		Note     *string           `excel:"header=Note"`
	}
	type config struct {
		Rows []row `excel:"key=rows;workflow=all;format=slice"`
	}
	plan, err := Compile[config](WithTrimTitle(true), WithCaseInsensitiveTitle(true))
	if err != nil {
		t.Fatal(err)
	}
	want := config{Rows: []row{{Flag: true, Signed: -2, Unsigned: 5, Ratio: 1.5, Labels: map[string]string{"a": "b"}, Note: stringPointer("memo")}}}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got config
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v", got)
	}
}

func TestSingleAndStructCodecs(t *testing.T) {
	type item struct {
		Name string `excel:"header=Name"`
	}
	type config struct {
		Title string `excel:"key=title;workflow=index;start_row=1;end_row=1;format=single"`
		Item  item   `excel:"key=item;workflow=start;start_row=2;format=struct"`
	}
	plan, err := Compile[config](WithUnconsumedRowsPolicy(IgnoreUnconsumedRows))
	if err != nil {
		t.Fatal(err)
	}
	want := config{Title: "Catalog", Item: item{Name: "Chair"}}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	var got config
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, sheet = %#v", got, sheet.Values())
	}
}

func TestInvalidMainInputs(t *testing.T) {
	plan, _ := Compile[roundCatalog]()
	var got roundCatalog
	if err := plan.Decode(context.Background(), Sheet{}, &got); !errors.Is(err, ErrSheetEmpty) {
		t.Fatalf("error = %v", err)
	}
	if err := plan.Decode(context.Background(), NewSheet("x", [][]string{{"x"}}), nil); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("error = %v", err)
	}
	if _, err := plan.Encode(context.Background(), nil); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("error = %v", err)
	}
}

func stringPointer(value string) *string { return &value }
