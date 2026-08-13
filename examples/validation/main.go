package main

import (
	"context"
	"errors"
	"fmt"

	mapper "github.com/wangw82/excel-struct-mapper"
)

type validatedRow struct {
	Count int `excel:"header=Count;required=true;validate=positive"`
}

func (r *validatedRow) ValidateMapping(context.Context) error {
	if r.Count > 100 {
		return errors.New("count must not exceed 100")
	}
	return nil
}

type validatedModel struct {
	Rows []validatedRow `excel:"key=rows;workflow=all;format=slice"`
}

func main() {
	fieldValidation := func(issue mapper.ValidationIssue) error {
		if issue.Rule == "positive" && issue.Value == "0" {
			return errors.New("count must be positive")
		}
		return nil
	}
	modelValidation := func(_ context.Context, value any) error {
		if len(value.(*validatedModel).Rows) == 0 {
			return errors.New("at least one row is required")
		}
		return nil
	}
	plan, err := mapper.Compile[validatedModel](
		mapper.WithValidation(fieldValidation),
		mapper.WithModelValidation(modelValidation),
	)
	must(err)

	var valid validatedModel
	validSheet := mapper.NewSheet("Data", [][]string{
		{"Count"},
		{"7"},
	})
	must(plan.Decode(context.Background(), validSheet, &valid))

	var invalid validatedModel
	fieldInvalidSheet := mapper.NewSheet("Data", [][]string{
		{"Count"},
		{"0"},
	})
	err = plan.Decode(context.Background(), fieldInvalidSheet, &invalid)
	var mapped *mapper.Error
	if !errors.As(err, &mapped) || mapped.Kind != mapper.KindValidation || mapped.Cell != "A2" {
		panic(fmt.Sprintf("unexpected field validation error: %v", err))
	}
	fieldCell := mapped.Cell

	mappingInvalidSheet := mapper.NewSheet("Data", [][]string{
		{"Count"},
		{"101"},
	})
	err = plan.Decode(context.Background(), mappingInvalidSheet, &invalid)
	if !errors.As(err, &mapped) || mapped.Kind != mapper.KindValidation {
		panic(fmt.Sprintf("unexpected mapping validation error: %v", err))
	}
	mappingCell := mapped.Cell

	if _, err := plan.Encode(context.Background(), validatedModel{}); err == nil {
		panic("model validation did not run")
	}
	fmt.Printf("validation: field at %s, mapped value at %s, whole model\n", fieldCell, mappingCell)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
