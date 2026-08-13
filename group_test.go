package mapper

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type sectionEntry struct {
	ID int `excel:"header=ID;required=true"`
}

type sectionPolicy struct {
	Limit int `excel:"header=Limit;required=true"`
}

type compoundSection struct {
	Entries  []sectionEntry  `excel:"key=entries;workflow=title;title=ITEMS;format=slice;min_rows=2;blank_line=true"`
	Policies []sectionPolicy `excel:"key=policies;workflow=title;title=POLICIES;format=slice;min_rows=2;blank_line=true"`
}

func TestRepeatedGroupRoundTrip(t *testing.T) {
	type model struct {
		Groups []*compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;format=group"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	want := model{Groups: []*compoundSection{
		{Entries: []sectionEntry{{ID: 101}}, Policies: []sectionPolicy{{Limit: 10}}},
		{Entries: []sectionEntry{{ID: 202}}, Policies: []sectionPolicy{{Limit: 20}}},
	}}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	wantRows := [][]string{
		{"ITEMS"}, {"ID"}, {"101"}, {},
		{"POLICIES"}, {"Limit"}, {"10"}, {},
		{"ITEMS"}, {"ID"}, {"202"}, {},
		{"POLICIES"}, {"Limit"}, {"20"}, {},
	}
	if !reflect.DeepEqual(sheet.Values(), NewSheet("", wantRows).Values()) {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got model
	if err := plan.Decode(context.Background(), NewSheet("Data", wantRows), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestRepeatedGroupRequiresSliceAndConsumesItsScope(t *testing.T) {
	type singleModel struct {
		Group compoundSection `excel:"key=group;workflow=repeat_title;title=ITEMS;format=group"`
	}
	if _, err := Compile[singleModel](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("non-slice repeated group error = %v", err)
	}

	type incompleteModel struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;format=group"`
	}
	incomplete, err := Compile[incompleteModel]()
	if err != nil {
		t.Fatal(err)
	}
	var incompleteGot incompleteModel
	err = incomplete.Decode(context.Background(), NewSheet("Data", [][]string{{"ITEMS"}, {"ID"}, {"101"}, {}, {"UNMAPPED"}}), &incompleteGot)
	if err == nil {
		t.Fatal("incomplete group accepted")
	}
}

func TestSingleGroupMergesRepeatedChildBlocks(t *testing.T) {
	type repeatedChildren struct {
		Entries  []sectionEntry  `excel:"key=entries;workflow=repeat_title;title=ITEMS;format=slice;blank_line=true"`
		Policies []sectionPolicy `excel:"key=policies;workflow=repeat_title;title=POLICIES;format=slice;blank_line=true"`
	}
	type model struct {
		Group repeatedChildren `excel:"key=group;workflow=all;format=group"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	sheet := NewSheet("Data", [][]string{
		{"ITEMS"}, {"ID"}, {"101"}, {},
		{"POLICIES"}, {"Limit"}, {"10"}, {},
		{"ITEMS"}, {"ID"}, {"202"}, {},
		{"POLICIES"}, {"Limit"}, {"20"}, {},
	})
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	want := model{Group: repeatedChildren{
		Entries:  []sectionEntry{{ID: 101}, {ID: 202}},
		Policies: []sectionPolicy{{Limit: 10}, {Limit: 20}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestRepeatedSliceMergesBlocksInSourceOrder(t *testing.T) {
	type model struct {
		Entries []sectionEntry `excel:"key=entries;workflow=repeat_title;title=ITEMS;format=slice;blank_line=true"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	sheet := NewSheet("Data", [][]string{{"ITEMS"}, {"ID"}, {"101"}, {}, {"ITEMS"}, {"ID"}, {"202"}, {}})
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	want := model{Entries: []sectionEntry{{ID: 101}, {ID: 202}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestEmptyRepeatedGroupIsRejectedDuringEncoding(t *testing.T) {
	type model struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;format=group"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Encode(context.Background(), model{}); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("empty group error = %v", err)
	}
}

func TestRangeGroupKeepsFollowingSiblingBlock(t *testing.T) {
	type model struct {
		Group compoundSection `excel:"key=group;workflow=title_range;title=ITEMS;end_title=POLICIES;format=group"`
		Tail  []sectionEntry  `excel:"key=tail;workflow=title;title=TAIL;format=slice;blank_line=true"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	want := model{
		Group: compoundSection{Entries: []sectionEntry{{ID: 101}}, Policies: []sectionPolicy{{Limit: 10}}},
		Tail:  []sectionEntry{{ID: 999}},
	}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	wantRows := [][]string{{"ITEMS"}, {"ID"}, {"101"}, {}, {"POLICIES"}, {"Limit"}, {"10"}, {}, {"TAIL"}, {"ID"}, {"999"}, {}}
	if !reflect.DeepEqual(sheet.Values(), NewSheet("", wantRows).Values()) {
		t.Fatalf("sheet = %#v", sheet.Values())
	}
	var got model
	if err := plan.Decode(context.Background(), NewSheet("Data", wantRows), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestRangeGroupAllowsFinalChildAtEndOfSheet(t *testing.T) {
	type model struct {
		Group compoundSection `excel:"key=group;workflow=title_range;title=ITEMS;end_title=POLICIES;format=group"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	want := model{Group: compoundSection{
		Entries:  []sectionEntry{{ID: 101}},
		Policies: []sectionPolicy{{Limit: 10}},
	}}
	sheet := NewSheet("Data", [][]string{
		{"ITEMS"}, {"ID"}, {"101"}, {},
		{"POLICIES"}, {"Limit"}, {"10"},
	})
	var got model
	if err := plan.Decode(context.Background(), sheet, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestRepeatedGroupStopsBeforeEndTitle(t *testing.T) {
	type model struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;end_title=TAIL;format=group"`
		Tail   []sectionEntry    `excel:"key=tail;workflow=title;title=TAIL;format=slice;blank_line=true"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	wantRows := [][]string{
		{"ITEMS"}, {"ID"}, {"101"}, {}, {"POLICIES"}, {"Limit"}, {"10"}, {},
		{"ITEMS"}, {"ID"}, {"202"}, {}, {"POLICIES"}, {"Limit"}, {"20"}, {},
		{"TAIL"}, {"ID"}, {"999"}, {},
	}
	var got model
	if err := plan.Decode(context.Background(), NewSheet("Data", wantRows), &got); err != nil {
		t.Fatal(err)
	}
	want := model{
		Groups: []compoundSection{
			{Entries: []sectionEntry{{ID: 101}}, Policies: []sectionPolicy{{Limit: 10}}},
			{Entries: []sectionEntry{{ID: 202}}, Policies: []sectionPolicy{{Limit: 20}}},
		},
		Tail: []sectionEntry{{ID: 999}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestNestedGroupRoundTrip(t *testing.T) {
	type nestedSection struct {
		Inner compoundSection `excel:"key=inner;workflow=title_range;title=ITEMS;end_title=POLICIES;format=group"`
	}
	type model struct {
		Outer nestedSection `excel:"key=outer;workflow=all;format=group"`
	}
	want := model{Outer: nestedSection{Inner: compoundSection{
		Entries:  []sectionEntry{{ID: 101}},
		Policies: []sectionPolicy{{Limit: 10}},
	}}}
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

func TestTitleGroupRoundTrip(t *testing.T) {
	type singleSection struct {
		Entries []sectionEntry `excel:"key=entries;workflow=title;title=ITEMS;format=slice;blank_line=true"`
	}
	type model struct {
		Group singleSection `excel:"key=group;workflow=title;title=ITEMS;format=group;blank_line=true"`
	}
	want := model{Group: singleSection{Entries: []sectionEntry{{ID: 101}}}}
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

func TestCustomWorkflowPreservesMultipleGroupBoundaries(t *testing.T) {
	const workflowName WorkflowName = "paired_sections"
	workflow := BlockWorkflowFunc{
		SelectFunc: func(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
			return selectRepeatTitle(ctx, sheet, request)
		},
		PlaceFunc: func(_ context.Context, output BlockOutput, _ BlockRequest, blocks [][]Line) error {
			if len(blocks) != 2 {
				return errors.New("expected two logical blocks")
			}
			for _, lines := range blocks {
				if err := output.PlaceLines(output.Len(), lines...); err != nil {
					return err
				}
			}
			return nil
		},
	}
	registry := NewRegistry()
	if err := registry.RegisterBlockWorkflow(workflowName, workflow); err != nil {
		t.Fatal(err)
	}
	type model struct {
		Groups []compoundSection `excel:"key=groups;workflow=paired_sections;title=ITEMS;format=group"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	want := model{Groups: []compoundSection{
		{Entries: []sectionEntry{{ID: 101}}, Policies: []sectionPolicy{{Limit: 10}}},
		{Entries: []sectionEntry{{ID: 202}}, Policies: []sectionPolicy{{Limit: 20}}},
	}}
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

func TestCustomWorkflowRejectsOverlappingOrOutOfOrderBlocks(t *testing.T) {
	for _, test := range []struct {
		name   string
		blocks []Block
	}{
		{"overlap", []Block{{StartRow: 0, EndRow: 2}, {StartRow: 1, EndRow: 3}}},
		{"out_of_order", []Block{{StartRow: 2, EndRow: 3}, {StartRow: 0, EndRow: 1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			workflow := BlockWorkflowFunc{
				SelectFunc: func(_ context.Context, sheet Sheet, _ BlockRequest) ([]Block, error) {
					blocks := append([]Block(nil), test.blocks...)
					for i := range blocks {
						blocks[i].Rows = sheet.Rows[blocks[i].StartRow:blocks[i].EndRow]
					}
					return blocks, nil
				},
				PlaceFunc: func(context.Context, BlockOutput, BlockRequest, [][]Line) error { return nil },
			}
			if err := registry.RegisterBlockWorkflow("invalid_multi", workflow); err != nil {
				t.Fatal(err)
			}
			type model struct {
				Rows []sectionEntry `excel:"key=rows;workflow=invalid_multi;format=slice"`
			}
			plan, err := Compile[model](WithRegistry(registry))
			if err != nil {
				t.Fatal(err)
			}
			var got model
			err = plan.Decode(context.Background(), NewSheet("Data", [][]string{{"ID"}, {"1"}, {"ID"}}), &got)
			if !errors.Is(err, ErrInvalidBlock) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRepeatedGroupRequiresFollowingBoundaryBlock(t *testing.T) {
	type model struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;end_title=TAIL;format=group"`
	}
	if _, err := Compile[model](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("boundary block error = %v", err)
	}
}

func TestRepeatedGroupRequiresConfiguredEndTitleInInput(t *testing.T) {
	type model struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;end_title=TAIL;format=group"`
		Tail   []sectionEntry    `excel:"key=tail;workflow=title;title=TAIL;format=slice;blank_line=true"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	var got model
	err = plan.Decode(context.Background(), NewSheet("Data", [][]string{{"ITEMS"}, {"ID"}, {"1"}, {}, {"POLICIES"}, {"Limit"}, {"10"}, {}}), &got)
	if err == nil {
		t.Fatal("missing group end title accepted")
	}
}

func TestNestedGroupErrorKeepsWorksheetCoordinates(t *testing.T) {
	type model struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=ITEMS;format=group"`
	}
	plan, err := Compile[model]()
	if err != nil {
		t.Fatal(err)
	}
	sheet := NewSheet("Data", [][]string{
		{"ITEMS"}, {"ID"}, {"101"}, {},
		{"POLICIES"}, {"Limit"}, {"bad"}, {},
	})
	var got model
	err = plan.Decode(context.Background(), sheet, &got)
	var mapped *Error
	if !errors.As(err, &mapped) || mapped.Cell != "A7" || mapped.Block != "policies" || mapped.Field != "Limit" {
		t.Fatalf("error = %#v", err)
	}
}

func TestCompileRejectsInvalidOrRecursiveGroup(t *testing.T) {
	type scalarGroup struct {
		Value int `excel:"key=value;workflow=all;format=group"`
	}
	if _, err := Compile[scalarGroup](); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("scalar group error = %v", err)
	}
	type recursiveGroup struct {
		Children []recursiveGroup `excel:"key=children;workflow=all;format=group"`
	}
	if _, err := Compile[recursiveGroup](); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("recursive group error = %v", err)
	}
	type mismatchedRange struct {
		Group compoundSection `excel:"key=group;workflow=title_range;title=BEGIN;end_title=END;format=group"`
	}
	if _, err := Compile[mismatchedRange](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("group range error = %v", err)
	}
	type noRangeDelimiterChild struct {
		Entries  []sectionEntry  `excel:"key=entries;workflow=title;title=ITEMS;format=slice;blank_line=true"`
		Policies []sectionPolicy `excel:"key=policies;workflow=title;title=POLICIES;format=slice"`
	}
	type noRangeDelimiter struct {
		Group noRangeDelimiterChild `excel:"key=group;workflow=title_range;title=ITEMS;end_title=POLICIES;format=group"`
	}
	if _, err := Compile[noRangeDelimiter](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("group range delimiter error = %v", err)
	}
	type mismatchedTitle struct {
		Groups []compoundSection `excel:"key=groups;workflow=repeat_title;title=OTHER;format=group"`
	}
	if _, err := Compile[mismatchedTitle](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("group title error = %v", err)
	}
	type unboundedSlice struct {
		Groups []compoundSection `excel:"key=groups;workflow=all;format=group"`
	}
	if _, err := Compile[unboundedSlice](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("group slice workflow error = %v", err)
	}
	type repeatedStruct struct {
		Value compoundSection `excel:"key=value;workflow=repeat_title;title=ITEMS;format=struct"`
	}
	if _, err := Compile[repeatedStruct](); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("repeated struct error = %v", err)
	}
}
