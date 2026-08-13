package mapper

import (
	"context"
	"fmt"
)

type standardWorkflow struct {
	name         WorkflowName
	selectBlocks func(context.Context, Sheet, BlockRequest) ([]Block, error)
}

func (w standardWorkflow) Select(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	return w.selectBlocks(ctx, sheet, request)
}

func (w standardWorkflow) Place(ctx context.Context, output BlockOutput, request BlockRequest, blocks [][]Line) error {
	if len(blocks) == 0 {
		return nil
	}
	if len(blocks) > 1 && w.name != WorkflowRepeatTitle {
		return fmt.Errorf("%w: workflow %q cannot place multiple blocks", ErrLayoutConflict, w.name)
	}
	for _, lines := range blocks {
		if err := w.placeBlock(ctx, output, request, lines); err != nil {
			return err
		}
	}
	return nil
}

func (w standardWorkflow) placeBlock(ctx context.Context, output BlockOutput, request BlockRequest, lines []Line) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	framed := make([]Line, 0, len(lines)+2)
	if request.Title != "" && !request.unframed {
		framed = append(framed, Line{request.Title})
	}
	framed = append(framed, lines...)
	if w.name == WorkflowTitleRange && request.EndTitle != "" && !request.unframed {
		framed = append(framed, Line{request.EndTitle})
	} else if request.BlankLine && !request.unframed {
		framed = append(framed, Line{})
	}
	contentRows := len(framed)
	if request.BlankLine && w.name != WorkflowTitleRange && !request.unframed {
		contentRows--
	}
	if w.name == WorkflowTitleRange && request.EndTitle != "" && !request.unframed {
		contentRows--
	}
	if contentRows < request.MinRows {
		return fmt.Errorf("%w: block has %d rows, minimum is %d", ErrLayoutConflict, contentRows, request.MinRows)
	}

	switch w.name {
	case WorkflowIndex:
		capacity := request.EndRow - request.StartRow + 1
		if len(framed) > capacity {
			return fmt.Errorf("%w: index block needs %d rows but range has %d", ErrLayoutConflict, len(framed), capacity)
		}
		for len(framed) < capacity {
			framed = append(framed, Line{})
		}
		return output.PlaceLines(request.StartRow, framed...)
	case WorkflowStart:
		return output.PlaceLines(request.StartRow, framed...)
	case WorkflowAll:
		return output.PlaceLines(0, framed...)
	default:
		start := output.Len()
		if request.StartRow > start {
			start = request.StartRow
		}
		return output.PlaceLines(start, framed...)
	}
}

func standardBlockWorkflow(name WorkflowName, selectBlocks func(context.Context, Sheet, BlockRequest) ([]Block, error)) BlockWorkflow {
	return standardWorkflow{name: name, selectBlocks: selectBlocks}
}

func builtinWorkflows() map[WorkflowName]BlockWorkflow {
	return map[WorkflowName]BlockWorkflow{
		WorkflowAll:         standardBlockWorkflow(WorkflowAll, selectAll),
		WorkflowIndex:       standardBlockWorkflow(WorkflowIndex, selectIndex),
		WorkflowStart:       standardBlockWorkflow(WorkflowStart, selectStart),
		WorkflowTitle:       standardBlockWorkflow(WorkflowTitle, selectTitle),
		WorkflowRepeatTitle: standardBlockWorkflow(WorkflowRepeatTitle, selectRepeatTitle),
		WorkflowTitleRange:  standardBlockWorkflow(WorkflowTitleRange, selectTitleRange),
	}
}

func selectAll(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	return selectOne(ctx, sheet, 0, len(sheet.Rows), request)
}

func selectIndex(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	if request.Optional && (request.StartRow >= len(sheet.Rows) || request.EndRow >= len(sheet.Rows)) {
		return nil, nil
	}
	return selectOne(ctx, sheet, request.StartRow, request.EndRow+1, request)
}

func selectStart(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	if request.Optional && request.StartRow >= len(sheet.Rows) {
		return nil, nil
	}
	return selectOne(ctx, sheet, request.StartRow, len(sheet.Rows), request)
}

func selectTitle(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	start := findTitle(sheet.Rows, request.Title, 0)
	if start < 0 {
		if request.Optional {
			return nil, nil
		}
		return nil, fmt.Errorf("title %q not found", request.Title)
	}
	end := len(sheet.Rows)
	if request.BlankLine {
		end = findBlank(sheet.Rows, start+1)
	}
	block, err := finishSelection(ctx, sheet, start, end, request)
	if err != nil {
		return nil, err
	}
	if request.BlankLine && end < len(sheet.Rows) {
		block.ConsumedEndRow = end + 1
	}
	return []Block{block}, nil
}

func selectRepeatTitle(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	var blocks []Block
	boundary := len(sheet.Rows)
	if request.EndTitle != "" {
		boundary = findTitle(sheet.Rows, request.EndTitle, request.StartRow)
		if boundary < 0 {
			if request.Optional && findTitle(sheet.Rows, request.Title, request.StartRow) < 0 {
				return nil, nil
			}
			return nil, fmt.Errorf("end title %q not found", request.EndTitle)
		}
	}
	for start := findTitleBefore(sheet.Rows, request.Title, request.StartRow, boundary); start >= 0; {
		next := findTitleBefore(sheet.Rows, request.Title, start+1, boundary)
		end := next
		if request.BlankLine {
			end = findBlank(sheet.Rows, start+1)
			if end > boundary {
				end = boundary
			}
		}
		if end < 0 {
			end = boundary
		}
		block, err := finishSelection(ctx, sheet, start, end, request)
		if err != nil {
			return nil, err
		}
		if request.BlankLine && end < len(sheet.Rows) {
			block.ConsumedEndRow = end + 1
		}
		blocks = append(blocks, block)
		if next < 0 {
			break
		}
		start = next
	}
	if len(blocks) == 0 {
		if request.Optional {
			return nil, nil
		}
		return nil, fmt.Errorf("title %q not found", request.Title)
	}
	return blocks, nil
}

func selectTitleRange(ctx context.Context, sheet Sheet, request BlockRequest) ([]Block, error) {
	start := 0
	if request.Title != "" {
		start = findTitle(sheet.Rows, request.Title, 0)
		if start < 0 {
			if request.Optional {
				return nil, nil
			}
			return nil, fmt.Errorf("title %q not found", request.Title)
		}
	}
	end := len(sheet.Rows)
	if request.EndTitle != "" {
		end = findTitle(sheet.Rows, request.EndTitle, start+1)
		if end < 0 {
			return nil, fmt.Errorf("end title %q not found", request.EndTitle)
		}
	}
	contentEnd := end
	if request.includeEndBlock && request.EndTitle != "" {
		contentEnd = findBlank(sheet.Rows, end+1)
	}
	block, err := finishSelection(ctx, sheet, start, contentEnd, request)
	if err != nil {
		return nil, err
	}
	if !request.includeEndBlock && request.EndTitle != "" {
		block.ConsumedEndRow = end + 1
	}
	return []Block{block}, nil
}

func selectOne(ctx context.Context, sheet Sheet, start, end int, request BlockRequest) ([]Block, error) {
	block, err := finishSelection(ctx, sheet, start, end, request)
	if err != nil {
		return nil, err
	}
	return []Block{block}, nil
}

func finishSelection(ctx context.Context, sheet Sheet, start, end int, request BlockRequest) (Block, error) {
	if err := ctx.Err(); err != nil {
		return Block{}, err
	}
	if start < 0 || end < start || end > len(sheet.Rows) {
		return Block{}, fmt.Errorf("invalid row range %d:%d", start+1, end)
	}
	minimum := request.MinRows
	if request.Optional && rangeContainsNoData(sheet.Rows, start, end, request) {
		minimum = 0
	}
	if end-start < minimum {
		return Block{}, fmt.Errorf("block has %d rows, minimum is %d", end-start, request.MinRows)
	}
	rows := make(Rows, end-start)
	copy(rows, sheet.Rows[start:end])
	return Block{Rows: NormalizeRows(rows), StartRow: start, EndRow: end}, nil
}

func rangeContainsNoData(rows Rows, start, end int, request BlockRequest) bool {
	if request.Title != "" {
		start++
	}
	for row := start; row < end && row < len(rows); row++ {
		if !rows[row].Empty() {
			return false
		}
	}
	return true
}

func findTitle(rows Rows, title string, from int) int {
	return findTitleBefore(rows, title, from, len(rows))
}

func findTitleBefore(rows Rows, title string, from, before int) int {
	for i := max(from, 0); i < min(before, len(rows)); i++ {
		if len(rows[i]) > 0 && rows[i][0].Value == title {
			return i
		}
	}
	return -1
}

func findBlank(rows Rows, from int) int {
	for i := from; i < len(rows); i++ {
		if rows[i].Empty() {
			return i
		}
	}
	return len(rows)
}
