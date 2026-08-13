package mapper

import "fmt"

type BlockRequest struct {
	StartRow        int
	EndRow          int
	Title           string
	EndTitle        string
	MinRows         int
	BlankLine       bool
	Optional        bool
	DataRow         int
	LabelCol        int
	ValueCol        int
	Label           string
	includeEndBlock bool
	unframed        bool
}

type Block struct {
	// Rows contains only the content passed to a block codec.
	Rows Rows
	// StartRow and EndRow describe the half-open content range.
	StartRow int
	EndRow   int
	// ConsumedEndRow optionally extends the consumed range to include trailing
	// structural delimiters. Zero means EndRow.
	ConsumedEndRow int
}

func validateBlock(sheet Sheet, block Block) error {
	if block.StartRow < 0 || block.EndRow < block.StartRow || block.EndRow > len(sheet.Rows) {
		return fmt.Errorf("%w: range %d:%d", ErrInvalidBlock, block.StartRow, block.EndRow)
	}
	if len(block.Rows) != block.EndRow-block.StartRow {
		return fmt.Errorf("%w: range contains %d rows, got %d", ErrInvalidBlock, block.EndRow-block.StartRow, len(block.Rows))
	}
	consumedEnd := block.ConsumedEndRow
	if consumedEnd == 0 {
		consumedEnd = block.EndRow
	}
	if consumedEnd < block.EndRow || consumedEnd > len(sheet.Rows) {
		return fmt.Errorf("%w: consumed end row %d", ErrInvalidBlock, consumedEnd)
	}
	for _, row := range block.Rows {
		for _, cell := range row {
			startRow := sheet.absoluteRow(block.StartRow)
			endRow := sheet.absoluteRow(block.EndRow)
			if cell.Row < startRow || cell.Row >= endRow {
				return fmt.Errorf("%w: cell row %d is outside range %d:%d", ErrInvalidBlock, cell.Row, block.StartRow, block.EndRow)
			}
		}
	}
	return nil
}
