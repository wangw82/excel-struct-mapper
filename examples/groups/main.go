package main

import (
	"context"
	"fmt"
	"reflect"

	mapper "github.com/wangw82/excel-struct-mapper"
)

type item struct {
	ID int `excel:"header=ID;required=true"`
}

type rule struct {
	Limit int `excel:"header=Limit;required=true"`
}

type section struct {
	Items []item `excel:"key=items;workflow=title;title=ITEMS;format=slice;blank_line=true"`
	Rules []rule `excel:"key=rules;workflow=title;title=RULES;format=slice;blank_line=true"`
}

type repeatedGroups struct {
	Groups []*section `excel:"key=groups;workflow=repeat_title;title=ITEMS;format=group"`
}

type rangeGroup struct {
	Section section `excel:"key=section;workflow=title_range;title=ITEMS;end_title=RULES;format=group"`
	Tail    []item  `excel:"key=tail;workflow=title;title=TAIL;format=slice;blank_line=true"`
}

type nestedSection struct {
	Inner section `excel:"key=inner;workflow=title_range;title=ITEMS;end_title=RULES;format=group"`
}

type nestedGroup struct {
	Outer nestedSection `excel:"key=outer;workflow=all;format=group"`
}

func main() {
	ctx := context.Background()
	mustRoundTrip(ctx, repeatedGroups{Groups: []*section{
		{Items: []item{{ID: 1}}, Rules: []rule{{Limit: 10}}},
		{Items: []item{{ID: 2}}, Rules: []rule{{Limit: 20}}},
	}})
	mustRoundTrip(ctx, rangeGroup{
		Section: section{Items: []item{{ID: 3}}, Rules: []rule{{Limit: 30}}},
		Tail:    []item{{ID: 99}},
	})
	mustRoundTrip(ctx, nestedGroup{Outer: nestedSection{Inner: section{
		Items: []item{{ID: 4}}, Rules: []rule{{Limit: 40}},
	}}})
	fmt.Println("groups: repeated compound blocks, title ranges, nested recursion")
}

func mustRoundTrip[T any](ctx context.Context, want T) {
	plan, err := mapper.Compile[T]()
	must(err)
	sheet, err := plan.Encode(ctx, want)
	must(err)
	var got T
	must(plan.Decode(ctx, sheet, &got))
	if !reflect.DeepEqual(got, want) {
		panic(fmt.Sprintf("round trip mismatch: got %#v, want %#v", got, want))
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
