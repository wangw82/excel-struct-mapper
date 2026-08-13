package mapper

import (
	"context"
	"fmt"
)

// BlockOutput is the write-side dependency used by BlockWorkflow. Implementors
// place lines without depending on a workbook format or a concrete table type.
type BlockOutput interface {
	Len() int
	PlaceLines(int, ...Line) error
}

// BlockWorkflow owns the read and write layout of one logical worksheet block.
// Select returns blocks in source order; returning more than one block is valid
// for repeated layouts and requires a compatible slice or group plan.
type BlockWorkflow interface {
	Select(context.Context, Sheet, BlockRequest) ([]Block, error)
	Place(context.Context, BlockOutput, BlockRequest, [][]Line) error
}

type BlockWorkflowFunc struct {
	SelectFunc func(context.Context, Sheet, BlockRequest) ([]Block, error)
	PlaceFunc  func(context.Context, BlockOutput, BlockRequest, [][]Line) error
}

func (f BlockWorkflowFunc) Select(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	if f.SelectFunc == nil {
		return nil, fmt.Errorf("block workflow does not support selection")
	}
	return f.SelectFunc(ctx, sheet, request)
}

func (f BlockWorkflowFunc) Place(ctx context.Context, output BlockOutput, request BlockRequest, blocks [][]Line) error {
	if f.PlaceFunc == nil {
		return fmt.Errorf("block workflow does not support placement")
	}
	return f.PlaceFunc(ctx, output, request, blocks)
}
