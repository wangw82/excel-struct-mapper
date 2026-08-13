package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	mapper "github.com/wangw82/excel-struct-mapper"
)

type state int

const (
	stateUnknown state = iota
	stateReady
)

func (s state) MarshalText() ([]byte, error) {
	if s == stateReady {
		return []byte("ready"), nil
	}
	return []byte("unknown"), nil
}

func (s *state) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "ready":
		*s = stateReady
	case "unknown":
		*s = stateUnknown
	default:
		return fmt.Errorf("unknown state %q", text)
	}
	return nil
}

type typeRow struct {
	Name     string            `excel:"header=Name"`
	Enabled  bool              `excel:"header=Enabled"`
	Signed   int64             `excel:"header=Signed"`
	Unsigned uint32            `excel:"header=Unsigned"`
	Ratio    float64           `excel:"header=Ratio"`
	Created  time.Time         `excel:"header=Created"`
	Note     *string           `excel:"header=Note;allow_empty=true"`
	Labels   map[string]string `excel:"header=Labels"`
	Steps    []int             `excel:"header=Steps"`
	State    state             `excel:"header=State"`
}

type typeModel struct {
	Rows []typeRow `excel:"key=rows;workflow=all;format=slice"`
}

func main() {
	note := "memo"
	want := typeModel{Rows: []typeRow{{
		Name: "Alpha", Enabled: true, Signed: -7, Unsigned: 9, Ratio: 1.25,
		Created: time.Date(2026, time.August, 13, 8, 30, 0, 0, time.UTC),
		Note:    &note, Labels: map[string]string{"tier": "stable"}, Steps: []int{1, 2}, State: stateReady,
	}}}
	plan, err := mapper.Compile[typeModel]()
	must(err)
	sheet, err := plan.Encode(context.Background(), want)
	must(err)
	var got typeModel
	must(plan.Decode(context.Background(), sheet, &got))
	if !reflect.DeepEqual(got, want) {
		panic(fmt.Sprintf("round trip mismatch: got %#v, want %#v", got, want))
	}
	fmt.Println("types: scalars, time, pointer, JSON compounds, text marshalers")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
