package mapper

import (
	"errors"
	"testing"
)

type testProduct struct {
	ID   int    `excel:"header=ID;required=true"`
	Name string `excel:"header=Name;allow_empty=true"`
}

type testCatalog struct {
	Products []testProduct `excel:"key=products;workflow=title;title=Products;min_rows=2;format=slice"`
}

func TestTagEscapes(t *testing.T) {
	options, err := parseOptions(`header=A\;B\=C\\D;required=true`, fieldOptions)
	if err != nil {
		t.Fatal(err)
	}
	if options["header"] != `A;B=C\D` {
		t.Fatalf("header = %q", options["header"])
	}
}

func TestTagFailures(t *testing.T) {
	for _, tag := range []string{"header=A;header=B", "unknown=x", "required=maybe", "header"} {
		options, err := parseOptions(tag, fieldOptions)
		if err == nil && tag == "required=maybe" {
			_, err = optionBool(options, "required", false)
		}
		if !errors.Is(err, ErrInvalidTag) {
			t.Fatalf("tag %q error = %v", tag, err)
		}
	}
}

func TestCompilePlanAndFailures(t *testing.T) {
	plan, err := Compile[testCatalog]()
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.blocks) != 1 || plan.blocks[0].request.Title != "Products" || plan.blocks[0].fields[0].header != "ID" {
		t.Fatalf("plan = %#v", plan)
	}
	type duplicate struct {
		A []testProduct `excel:"key=same;workflow=all;format=slice"`
		B []testProduct `excel:"key=same;workflow=all;format=slice"`
	}
	if _, err := Compile[duplicate](); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("error = %v", err)
	}
	type unknown struct {
		A []testProduct `excel:"key=a;workflow=missing;format=slice"`
	}
	if _, err := Compile[unknown](); !errors.Is(err, ErrUnknownWorkflow) {
		t.Fatalf("error = %v", err)
	}
}

func TestUnknownTagKeysAreRejected(t *testing.T) {
	type row struct {
		Name string `excel:"header=Name"`
	}
	type unknownBlockOption struct {
		Rows []row `excel:"key=rows;unknown_block=x;format=slice"`
	}
	type unknownFieldOption struct {
		Rows []struct {
			Name string `excel:"header=Name;unknown_field=x"`
		} `excel:"key=rows;workflow=all;format=slice"`
	}
	for _, compile := range []func() error{
		func() error { _, err := Compile[unknownBlockOption](); return err },
		func() error { _, err := Compile[unknownFieldOption](); return err },
	} {
		if err := compile(); !errors.Is(err, ErrInvalidTag) {
			t.Fatalf("error = %v", err)
		}
	}
}
