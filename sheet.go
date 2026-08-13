package mapper

import (
	"fmt"
	"strings"
)

type Row []Cell

type Rows []Row

func (r Row) Empty() bool {
	for _, cell := range r {
		if strings.TrimSpace(cell.Value) != "" {
			return false
		}
	}
	return true
}

func (r Row) Cell(col int) (Cell, error) {
	if col < 0 || col >= len(r) {
		return Cell{}, ErrOutOfBounds
	}
	return r[col], nil
}

func NormalizeRows(rows Rows) Rows {
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	result := make(Rows, len(rows))
	for i, row := range rows {
		result[i] = make(Row, width)
		for j := 0; j < width; j++ {
			result[i][j] = Cell{Row: i, Col: j}
		}
		copy(result[i], row)
	}
	return result
}

type Sheet struct {
	Name      string
	Rows      Rows
	rowOffset int
}

func (s Sheet) scope(block Block) Sheet {
	return Sheet{Name: s.Name, Rows: s.Rows[block.StartRow:block.EndRow], rowOffset: s.rowOffset + block.StartRow}
}

func (s Sheet) absoluteRow(row int) int { return s.rowOffset + row }

func NewSheet(name string, values [][]string) Sheet {
	rows := make(Rows, len(values))
	for i, valuesRow := range values {
		rows[i] = make(Row, len(valuesRow))
		for j, value := range valuesRow {
			rows[i][j] = Cell{Row: i, Col: j, Value: value}
		}
	}
	return Sheet{Name: name, Rows: NormalizeRows(rows)}
}

func (s Sheet) Cell(row, col int) (Cell, error) {
	if row < 0 || row >= len(s.Rows) {
		return Cell{}, ErrOutOfBounds
	}
	return s.Rows[row].Cell(col)
}

func (s Sheet) Values() [][]string {
	values := make([][]string, len(s.Rows))
	for i, row := range s.Rows {
		values[i] = make([]string, len(row))
		for j, cell := range row {
			values[i][j] = cell.Value
		}
	}
	return values
}

type Line []string

type sheetOutput struct {
	Name     string
	Lines    []Line
	occupied []bool
}

func (t *sheetOutput) Len() int { return len(t.Lines) }

func (t *sheetOutput) PlaceLines(start int, lines ...Line) error {
	if start < 0 {
		return ErrLayoutConflict
	}
	end := start + len(lines)
	for len(t.Lines) < end {
		t.Lines = append(t.Lines, Line{})
	}
	for len(t.occupied) < end {
		t.occupied = append(t.occupied, false)
	}
	for row := start; row < end; row++ {
		if t.occupied[row] {
			return fmt.Errorf("%w: row %d is already occupied", ErrLayoutConflict, row+1)
		}
	}
	for i, line := range lines {
		t.Lines[start+i] = append(Line(nil), line...)
		t.occupied[start+i] = true
	}
	return nil
}

func (t sheetOutput) Sheet() Sheet {
	values := make([][]string, len(t.Lines))
	for i := range t.Lines {
		values[i] = append([]string(nil), t.Lines[i]...)
	}
	return NewSheet(t.Name, values)
}
