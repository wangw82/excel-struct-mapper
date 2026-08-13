package mapper

import (
	"context"
	"sync"
	"testing"
)

func TestRegistryBuildAndConcurrentRead(t *testing.T) {
	registry := NewRegistry()
	codec := ValueCodecFunc{DecodeFunc: builtinValueCodec{}.Decode, EncodeFunc: builtinValueCodec{}.Encode}
	if err := registry.RegisterValueCodec("custom", codec); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterValueCodec("custom", codec); err == nil {
		t.Fatal("duplicate registration succeeded")
	}
	built := registry.Build()
	if err := built.RegisterValueCodec("later", codec); err == nil {
		t.Fatal("built registry changed")
	}
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if built.valueCodecs["custom"] == nil {
				t.Error("value codec missing")
			}
		}()
	}
	group.Wait()
	workflow := standardBlockWorkflow("custom", selectAll)
	if err := registry.RegisterBlockWorkflow("custom", workflow); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterBlockWorkflow("custom", workflow); err == nil {
		t.Fatal("duplicate workflow accepted")
	}
}

func TestWorkflowSelectionFailures(t *testing.T) {
	sheet := NewSheet("Data", [][]string{{"A"}, {"B"}})
	for _, test := range []struct {
		name    WorkflowName
		request BlockRequest
	}{
		{WorkflowTitle, BlockRequest{Title: "Missing"}},
		{WorkflowRepeatTitle, BlockRequest{Title: "Missing"}},
		{WorkflowTitleRange, BlockRequest{Title: "A", EndTitle: "Missing"}},
		{WorkflowIndex, BlockRequest{StartRow: 3, EndRow: 4}},
	} {
		if _, err := builtinWorkflows()[test.name].Select(context.Background(), sheet, test.request); err == nil {
			t.Fatalf("%s accepted invalid request", test.name)
		}
	}
}

func TestBuiltinWorkflows(t *testing.T) {
	sheet := NewSheet("Data", [][]string{{"Start"}, {"H"}, {"1"}, {}, {"Start"}, {"H"}, {"2"}, {"End"}})
	tests := []struct {
		name       WorkflowName
		request    BlockRequest
		start, end int
		count      int
	}{
		{WorkflowAll, BlockRequest{}, 0, 8, 1},
		{WorkflowIndex, BlockRequest{StartRow: 1, EndRow: 2}, 1, 3, 1},
		{WorkflowStart, BlockRequest{StartRow: 4}, 4, 8, 1},
		{WorkflowTitle, BlockRequest{Title: "Start", BlankLine: true}, 0, 3, 1},
		{WorkflowRepeatTitle, BlockRequest{Title: "Start"}, 0, 4, 2},
		{WorkflowTitleRange, BlockRequest{Title: "Start", EndTitle: "End"}, 0, 7, 1},
	}
	for _, test := range tests {
		t.Run(string(test.name), func(t *testing.T) {
			blocks, err := builtinWorkflows()[test.name].Select(context.Background(), sheet, test.request)
			if err != nil || len(blocks) != test.count || blocks[0].StartRow != test.start || blocks[0].EndRow != test.end {
				t.Fatalf("blocks = %#v, %v", blocks, err)
			}
		})
	}
}

func TestWorkflowCancellationAndMinimum(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	workflow := builtinWorkflows()[WorkflowAll]
	if _, err := workflow.Select(ctx, NewSheet("Data", [][]string{{"x"}}), BlockRequest{}); err == nil {
		t.Fatal("cancellation ignored")
	}
	if _, err := workflow.Select(context.Background(), NewSheet("Data", [][]string{{"x"}}), BlockRequest{MinRows: 2}); err == nil {
		t.Fatal("minimum ignored")
	}
}
