package mapper

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestApplicationTagHandlersCompileAndRunRecursively(t *testing.T) {
	registry := NewRegistry()
	text := TagHandlerFunc(func(_ context.Context, tag TagContext) (any, error) {
		return map[string]any{
			"label": tag.Params,
			"field": tag.Field.Name,
			"json":  tag.Field.Tag.Get("json"),
		}, nil
	})
	group := TagHandlerFunc(func(_ context.Context, tag TagContext) (any, error) {
		return fmt.Sprintf("%s:%d", tag.Params, len(tag.Children)), nil
	})
	flag := TagHandlerFunc(func(_ context.Context, tag TagContext) (any, error) {
		return tag.Params == "", nil
	})
	for name, handler := range map[string]TagHandler{
		"text":  text,
		"group": group,
		"flag":  flag,
	} {
		if err := registry.RegisterTagHandler("ui", name, handler); err != nil {
			t.Fatal(err)
		}
	}

	type row struct {
		Name string `excel:"header=Name"`
	}
	type section struct {
		Title string `json:"title" ui:"text=Heading,Heading"`
	}
	type model struct {
		Rows    []row   `excel:"key=rows;workflow=all;format=slice"`
		Section section `ui:"group=Panel"`
		Enabled bool    `ui:"flag"`
		Ignored bool    `ui:"-"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}

	results, err := plan.RunTag(context.Background(), "ui")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	sectionResult := results[0]
	if sectionResult.Path != "Section" || sectionResult.Name != "group" || sectionResult.Params != "Panel" || sectionResult.Output != "Panel:1" {
		t.Fatalf("section = %#v", sectionResult)
	}
	if len(sectionResult.Children) != 1 || sectionResult.Children[0].Path != "Section.Title" {
		t.Fatalf("children = %#v", sectionResult.Children)
	}
	childOutput, ok := sectionResult.Children[0].Output.(map[string]any)
	if !ok || childOutput["label"] != "Heading,Heading" || childOutput["field"] != "Title" || childOutput["json"] != "title" {
		t.Fatalf("child output = %#v", sectionResult.Children[0].Output)
	}
	if results[1].Path != "Enabled" || results[1].Output != true {
		t.Fatalf("flag = %#v", results[1])
	}

	// A compiled plan owns resolved handlers and no longer consults the builder.
	if err := registry.RegisterTagHandler("ui", "later", flag); err != nil {
		t.Fatal(err)
	}
	second, err := plan.RunTag(context.Background(), "ui")
	if err != nil || !reflect.DeepEqual(second, results) {
		t.Fatalf("second run = %#v, %v", second, err)
	}
	var groupWait sync.WaitGroup
	for i := 0; i < 20; i++ {
		groupWait.Add(1)
		go func() {
			defer groupWait.Done()
			concurrent, runErr := plan.RunTag(context.Background(), "ui")
			if runErr != nil || !reflect.DeepEqual(concurrent, results) {
				t.Errorf("concurrent run = %#v, %v", concurrent, runErr)
			}
		}()
	}
	groupWait.Wait()
}

func TestApplicationTagHandlersRejectInvalidConfiguration(t *testing.T) {
	handler := TagHandlerFunc(func(context.Context, TagContext) (any, error) { return nil, nil })
	registry := NewRegistry()
	if err := registry.RegisterTagHandler("excel", "custom", handler); !errors.Is(err, ErrInvalidRegistry) {
		t.Fatalf("reserved tag error = %v", err)
	}
	if err := registry.RegisterTagHandler("bad tag", "custom", handler); !errors.Is(err, ErrInvalidRegistry) {
		t.Fatalf("invalid tag error = %v", err)
	}
	if err := registry.RegisterTagHandler("ui", "bad=name", handler); !errors.Is(err, ErrInvalidRegistry) {
		t.Fatalf("invalid handler name error = %v", err)
	}
	if err := registry.RegisterTagHandler("ui", "custom", handler); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterTagHandler("ui", "custom", handler); !errors.Is(err, ErrInvalidRegistry) {
		t.Fatalf("duplicate error = %v", err)
	}

	type row struct {
		Name string `excel:"header=Name"`
	}
	type unknownHandler struct {
		Rows  []row  `excel:"key=rows;workflow=all;format=slice"`
		Title string `ui:"missing=value"`
	}
	if _, err := Compile[unknownHandler](WithRegistry(registry)); !errors.Is(err, ErrUnknownTagHandler) {
		t.Fatalf("unknown handler error = %v", err)
	}
	type invalidTag struct {
		Rows  []row  `excel:"key=rows;workflow=all;format=slice"`
		Title string `ui:"=value"`
	}
	if _, err := Compile[invalidTag](WithRegistry(registry)); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("invalid value error = %v", err)
	}
}

func TestApplicationTagHandlerErrorsAndCancellation(t *testing.T) {
	want := errors.New("application failure")
	registry := NewRegistry()
	if err := registry.RegisterTagHandler("ui", "fail", TagHandlerFunc(func(context.Context, TagContext) (any, error) {
		return nil, want
	})); err != nil {
		t.Fatal(err)
	}
	type row struct {
		Name string `excel:"header=Name"`
	}
	type model struct {
		Rows  []row  `excel:"key=rows;workflow=all;format=slice"`
		Title string `ui:"fail=value"`
	}
	plan, err := Compile[model](WithRegistry(registry))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.RunTag(context.Background(), "missing"); !errors.Is(err, ErrUnknownTag) {
		t.Fatalf("unknown tag error = %v", err)
	}
	if _, err := plan.RunTag(context.Background(), "ui"); !errors.Is(err, want) {
		t.Fatalf("handler error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := plan.RunTag(ctx, "ui"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}
