package mapper

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDecodeErrorsAreStructuredAndAtomic(t *testing.T) {
	plan, _ := Compile[roundCatalog]()
	destination := roundCatalog{Products: []roundProduct{{ID: 9}}}
	err := plan.Decode(context.Background(), NewSheet("Catalog", [][]string{{"Products"}, {"ID", "Name"}, {"bad", "Chair"}}), &destination)
	var mapped *Error
	if !errors.As(err, &mapped) || mapped.Kind != KindConversion || mapped.Cell != "A3" || mapped.Field != "ID" {
		t.Fatalf("error = %#v", err)
	}
	if destination.Products[0].ID != 9 {
		t.Fatalf("destination changed = %#v", destination)
	}
}

func TestUnconsumedRowsPolicy(t *testing.T) {
	type config struct {
		Products []roundProduct `excel:"key=products;workflow=index;start_row=2;end_row=3;format=slice"`
	}
	sheet := NewSheet("Data", [][]string{{"orphan"}, {"ID", "Name"}, {"1", "Chair"}})
	plan, _ := Compile[config]()
	var got config
	if err := plan.Decode(context.Background(), sheet, &got); err == nil {
		t.Fatal("unconsumed row accepted")
	}
	plan, _ = Compile[config](WithUnconsumedRowsPolicy(IgnoreUnconsumedRows))
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
}

func TestValidationAndCancellation(t *testing.T) {
	validation := func(issue ValidationIssue) error {
		if issue.Rule == "positive" && issue.Value == "0" {
			return errors.New("must be positive")
		}
		return nil
	}
	type row struct {
		Count int `excel:"header=Count;validate=positive"`
	}
	type config struct {
		Rows []row `excel:"key=rows;workflow=all;format=slice"`
	}
	plan, _ := Compile[config](WithValidation(validation))
	var got config
	err := plan.Decode(context.Background(), NewSheet("Data", [][]string{{"Count"}, {"0"}}), &got)
	var mapped *Error
	if !errors.As(err, &mapped) || mapped.Kind != KindValidation {
		t.Fatalf("error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := plan.Decode(ctx, NewSheet("Data", [][]string{{"Count"}}), &got); !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v", err)
	}
}
