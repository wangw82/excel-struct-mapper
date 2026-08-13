package mapper

import (
	"context"
	"fmt"
	"reflect"
)

func (p *Plan) decodeForm(ctx context.Context, sheetName string, plan blockPlan, block Block) (reflect.Value, error) {
	item := reflect.New(plan.itemType).Elem()
	rows := make(map[string]Row, len(block.Rows))
	for rowIndex := plan.request.DataRow; rowIndex < len(block.Rows); rowIndex++ {
		row := block.Rows[rowIndex]
		if plan.request.LabelCol >= len(row) {
			continue
		}
		label := p.normalizeTitle(row[plan.request.LabelCol].Value)
		if label == "" {
			continue
		}
		if _, exists := rows[label]; exists {
			cell := row[plan.request.LabelCol]
			return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, "", cell.Row, cell.Col, fmt.Errorf("duplicate form label %q", cell.Value))
		}
		rows[label] = row
	}
	for _, field := range plan.fields {
		if field.skipDecode {
			continue
		}
		row, exists := rows[p.normalizeTitle(field.header)]
		if !exists {
			if field.required {
				return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, field.name, blockAbsoluteRow(block, 0), plan.request.LabelCol, fmt.Errorf("required label %q is missing", field.header))
			}
			continue
		}
		cells, err := blockValueCells(row, plan.request.ValueCol, field.multiCell, field.separator)
		if err != nil {
			return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, field.name, row[0].Row, plan.request.ValueCol, err)
		}
		if cellsEmpty(cells) {
			if field.required && !field.allowEmpty {
				return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, field.name, cells[0].Row, cells[0].Col, fmt.Errorf("value is required"))
			}
			continue
		}
		value, err := field.valueCodec.Decode(ctx, ValueCodecContext{Sheet: sheetName, Block: plan.key, Field: field.name, Cell: cells[0], Options: field.codecOptions}, cells, field.typeOf)
		if err != nil {
			return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, field.name, cells[0].Row, cells[0].Col, err)
		}
		if err := p.validateField(field, value, cells[0]); err != nil {
			return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, field.name, cells[0].Row, cells[0].Col, err)
		}
		item.FieldByIndex(field.index).Set(value)
	}
	return convertGroupItem(item, plan.typeOf)
}

func (p *Plan) decodeTranspose(ctx context.Context, sheetName string, plan blockPlan, block Block) (reflect.Value, error) {
	rows := make(map[string]Row, len(block.Rows))
	maxColumn := plan.request.ValueCol
	for rowIndex := plan.request.DataRow; rowIndex < len(block.Rows); rowIndex++ {
		row := block.Rows[rowIndex]
		if len(row) > maxColumn {
			maxColumn = len(row)
		}
		if plan.request.LabelCol < len(row) {
			label := p.normalizeTitle(row[plan.request.LabelCol].Value)
			if label != "" {
				if _, exists := rows[label]; exists {
					cell := row[plan.request.LabelCol]
					return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, "", cell.Row, cell.Col, fmt.Errorf("duplicate transpose label %q", cell.Value))
				}
				rows[label] = row
			}
		}
	}
	result := reflect.MakeSlice(plan.typeOf, 0, max(0, maxColumn-plan.request.ValueCol))
	for column := plan.request.ValueCol; column < maxColumn; column++ {
		if transposeColumnEmpty(rows, column) {
			continue
		}
		item := reflect.New(plan.itemType).Elem()
		for _, field := range plan.fields {
			if field.skipDecode {
				continue
			}
			row, exists := rows[p.normalizeTitle(field.header)]
			if !exists {
				if field.required {
					return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, field.name, blockAbsoluteRow(block, 0), plan.request.LabelCol, fmt.Errorf("required label %q is missing", field.header))
				}
				continue
			}
			cell := Cell{Row: row[0].Row, Col: column}
			if column < len(row) {
				cell = row[column]
			}
			if cell.Value == "" {
				if field.required && !field.allowEmpty {
					return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, field.name, cell.Row, cell.Col, fmt.Errorf("value is required"))
				}
				continue
			}
			value, err := field.valueCodec.Decode(ctx, ValueCodecContext{Sheet: sheetName, Block: plan.key, Field: field.name, Cell: cell, Options: field.codecOptions}, []Cell{cell}, field.typeOf)
			if err != nil {
				return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, field.name, cell.Row, cell.Col, err)
			}
			if err := p.validateField(field, value, cell); err != nil {
				return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, field.name, cell.Row, cell.Col, err)
			}
			item.FieldByIndex(field.index).Set(value)
		}
		converted, err := convertGroupItem(item, plan.typeOf.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		result = reflect.Append(result, converted)
	}
	return result, nil
}

func (p *Plan) encodeForm(ctx context.Context, plan blockPlan, value reflect.Value) ([]Line, error) {
	value, err := dereferenceMappedValue(value)
	if err != nil {
		return nil, err
	}
	lines := make([]Line, contentDataRow(plan))
	for _, field := range plan.fields {
		if field.skipEncode {
			continue
		}
		cells, err := field.valueCodec.Encode(ctx, ValueCodecContext{Block: plan.key, Field: field.name, Options: field.codecOptions}, value.FieldByIndex(field.index))
		if err != nil {
			return nil, locatedError(KindConversion, "", plan.key, field.name, -1, -1, err)
		}
		if !field.multiCell && len(cells) > 1 {
			return nil, fmt.Errorf("field %s requires multi_cell=true", field.name)
		}
		width := max(plan.request.LabelCol, plan.request.ValueCol+len(cells)-1) + 1
		line := make(Line, width)
		line[plan.request.LabelCol] = field.header
		copy(line[plan.request.ValueCol:], cells)
		if field.separator != "" {
			line = append(line, field.separator)
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (p *Plan) encodeTranspose(ctx context.Context, plan blockPlan, value reflect.Value) ([]Line, error) {
	items := value
	width := plan.request.ValueCol + items.Len()
	lines := make([]Line, contentDataRow(plan))
	for _, field := range plan.fields {
		if field.skipEncode {
			continue
		}
		line := make(Line, max(width, plan.request.LabelCol+1))
		line[plan.request.LabelCol] = field.header
		for i := 0; i < items.Len(); i++ {
			item, err := dereferenceMappedValue(items.Index(i))
			if err != nil {
				return nil, err
			}
			cells, err := field.valueCodec.Encode(ctx, ValueCodecContext{Block: plan.key, Field: field.name, Options: field.codecOptions}, item.FieldByIndex(field.index))
			if err != nil {
				return nil, locatedError(KindConversion, "", plan.key, field.name, -1, -1, err)
			}
			if len(cells) > 1 {
				return nil, fmt.Errorf("field %s cannot encode multiple cells in transpose format", field.name)
			}
			if len(cells) == 1 {
				line[plan.request.ValueCol+i] = cells[0]
			}
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (p *Plan) validateField(field fieldPlan, value reflect.Value, cell Cell) error {
	if field.validate == "" || p.validation == nil {
		return nil
	}
	return p.validation(ValidationIssue{Field: field.name, Rule: field.validate, Value: cell.Value})
}

func validateBlockLabel(p *Plan, plan blockPlan, row Row) error {
	if plan.request.Label == "" {
		return nil
	}
	if plan.request.LabelCol >= len(row) || p.normalizeTitle(row[plan.request.LabelCol].Value) != p.normalizeTitle(plan.request.Label) {
		return fmt.Errorf("label must be %q", plan.request.Label)
	}
	return nil
}

func blockValueCells(row Row, start int, multiple bool, separator string) ([]Cell, error) {
	if start < 0 || start >= len(row) {
		return nil, ErrOutOfBounds
	}
	end := start + 1
	if multiple {
		end = len(row)
		if separator != "" {
			for i := start; i < len(row); i++ {
				if row[i].Value == separator {
					end = i
					break
				}
			}
		}
	}
	cells := append([]Cell(nil), row[start:end]...)
	for len(cells) > 1 && cells[len(cells)-1].Value == "" {
		cells = cells[:len(cells)-1]
	}
	return cells, nil
}

func boundedFieldCells(row Row, start int, columns map[string]int, separator string) []Cell {
	end := len(row)
	if separator != "" {
		for i := start; i < len(row); i++ {
			if row[i].Value == separator {
				end = i
				break
			}
		}
	} else {
		for _, column := range columns {
			if column > start && column < end {
				end = column
			}
		}
	}
	cells := append([]Cell(nil), row[start:end]...)
	for len(cells) > 1 && cells[len(cells)-1].Value == "" {
		cells = cells[:len(cells)-1]
	}
	return cells
}

func cellsEmpty(cells []Cell) bool {
	for _, cell := range cells {
		if cell.Value != "" {
			return false
		}
	}
	return true
}

func transposeColumnEmpty(rows map[string]Row, column int) bool {
	for _, row := range rows {
		if column < len(row) && row[column].Value != "" {
			return false
		}
	}
	return true
}

func contentDataRow(plan blockPlan) int {
	row := plan.request.DataRow
	if plan.request.Title != "" && !plan.request.unframed {
		row--
	}
	return max(row, 0)
}

func dereferenceMappedValue(value reflect.Value) (reflect.Value, error) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, ErrInvalidSource
		}
		value = value.Elem()
	}
	return value, nil
}

func blockAbsoluteRow(block Block, relative int) int {
	if relative >= 0 && relative < len(block.Rows) && len(block.Rows[relative]) > 0 {
		return block.Rows[relative][0].Row
	}
	return block.StartRow + relative
}
