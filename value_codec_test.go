package mapper

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestCustomValueCodecRoundTrip(t *testing.T) {
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
	registry := NewRegistry()
	if err := registry.RegisterValueCodec("sku", codec); err != nil {
		t.Fatal(err)
	}
	type row struct {
		SKU string `excel:"header=SKU;value_codec=sku;codec_options=SKU-"`
	}
	type config struct {
		Rows []row `excel:"key=rows;workflow=all;format=slice"`
	}
	plan, err := Compile[config](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := plan.Encode(context.Background(), config{Rows: []row{{SKU: "9"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got config
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if got.Rows[0].SKU != "9" {
		t.Fatalf("got = %#v", got)
	}
}
