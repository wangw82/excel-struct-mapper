package main

import (
	"context"
	"fmt"
	"reflect"

	mapper "github.com/wangw82/excel-struct-mapper"
)

type tableRow struct {
	ID     int      `excel:"header=ID;required=true"`
	Name   string   `excel:"header=Name;allow_empty=true"`
	Labels []string `excel:"header=Labels;multi_cell=true;separator=/"`
}

type tableModel struct {
	Rows []tableRow `excel:"key=rows;workflow=all;format=slice"`
}

type profile struct {
	Name string `excel:"header=Name;required=true"`
	Tier int    `excel:"header=Tier;required=true"`
}

type structModel struct {
	Profile *profile `excel:"key=profile;workflow=title;title=PROFILE;format=struct;blank_line=true"`
}

type titledRow struct {
	ID int `excel:"header=ID"`
}

type repeatedModel struct {
	Optional []titledRow `excel:"key=optional;workflow=title;title=OPTIONAL;format=slice;blank_line=true;optional=true"`
	Rows     []titledRow `excel:"key=rows;workflow=repeat_title;title=RECORDS;format=slice;blank_line=true"`
}

type scalarModel struct {
	Limit int `excel:"key=limit;workflow=index;start_row=1;end_row=1;format=single;label=LIMIT;label_col=1;value_col=2"`
}

type metadata struct {
	Name   string   `excel:"header=Name;required=true"`
	Labels []string `excel:"header=Labels;multi_cell=true;separator=/"`
}

type formModel struct {
	Metadata metadata `excel:"key=metadata;workflow=title;title=META;format=form;blank_line=true"`
}

type transposedEntry struct {
	Region string `excel:"header=Region;required=true"`
	Limit  int    `excel:"header=Limit;required=true"`
}

type transposeModel struct {
	Entries []transposedEntry `excel:"key=entries;workflow=title;title=RECORDS;format=transpose;blank_line=true"`
}

type openRangeModel struct {
	Rows []titledRow `excel:"key=rows;workflow=title_range;title=BEGIN;format=slice"`
}

type leadingRangeModel struct {
	Rows []titledRow `excel:"key=rows;workflow=title_range;end_title=END;format=slice"`
}

type indexedModel struct {
	Rows []titledRow `excel:"key=rows;workflow=index;start_row=2;end_row=3;format=slice"`
}

func main() {
	ctx := context.Background()
	mustRoundTrip(ctx, tableModel{Rows: []tableRow{{ID: 1, Name: "Alpha", Labels: []string{"new", "featured"}}}})
	mustRoundTrip(ctx, structModel{Profile: &profile{Name: "Alpha", Tier: 2}})
	mustRoundTrip(ctx, scalarModel{Limit: 12})
	mustRoundTrip(ctx, formModel{Metadata: metadata{Name: "Alpha", Labels: []string{"stable", "public"}}})
	mustRoundTrip(ctx, transposeModel{Entries: []transposedEntry{{Region: "North", Limit: 10}, {Region: "South", Limit: 20}}})

	repeatedPlan := mustCompile[repeatedModel]()
	var repeated repeatedModel
	sheet := mapper.NewSheet("Data", [][]string{
		{"RECORDS"},
		{"ID"},
		{"1"},
		{},
		{"RECORDS"},
		{"ID"},
		{"2"},
		{},
	})
	must(repeatedPlan.Decode(ctx, sheet, &repeated))
	if !reflect.DeepEqual(repeated.Rows, []titledRow{{ID: 1}, {ID: 2}}) || repeated.Optional != nil {
		panic(fmt.Sprintf("unexpected repeated blocks: %#v", repeated))
	}

	openPlan := mustCompile[openRangeModel]()
	var open openRangeModel
	openSheet := mapper.NewSheet("Data", [][]string{
		{"BEGIN"},
		{"ID"},
		{"7"},
	})
	must(openPlan.Decode(ctx, openSheet, &open))
	if !reflect.DeepEqual(open.Rows, []titledRow{{ID: 7}}) {
		panic(fmt.Sprintf("unexpected open range: %#v", open))
	}
	leadingPlan := mustCompile[leadingRangeModel]()
	var leading leadingRangeModel
	leadingSheet := mapper.NewSheet("Data", [][]string{
		{"ID"},
		{"8"},
		{"END"},
	})
	must(leadingPlan.Decode(ctx, leadingSheet, &leading))
	if !reflect.DeepEqual(leading.Rows, []titledRow{{ID: 8}}) {
		panic(fmt.Sprintf("unexpected leading range: %#v", leading))
	}

	indexedPlan := mustCompile[indexedModel](mapper.WithUnconsumedRowsPolicy(mapper.IgnoreUnconsumedRows))
	var indexed indexedModel
	indexedSheet := mapper.NewSheet("Data", [][]string{
		{"IGNORED"},
		{"ID"},
		{"9"},
	})
	must(indexedPlan.Decode(ctx, indexedSheet, &indexed))
	if !reflect.DeepEqual(indexed.Rows, []titledRow{{ID: 9}}) {
		panic(fmt.Sprintf("unexpected indexed block: %#v", indexed))
	}

	fmt.Println("layouts: struct, slice, single, form, transpose, repeated, optional, multi-cell, ranges")
}

func mustRoundTrip[T any](ctx context.Context, want T) {
	plan := mustCompile[T]()
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
