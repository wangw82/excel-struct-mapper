package mapper

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func (p *Plan) Decode(ctx context.Context, sheet Sheet, destination any) error {
	return p.decode(ctx, sheet, destination)
}

func (p *Plan) decode(ctx context.Context, sheet Sheet, destination any) error {
	if p == nil {
		return ErrInvalidModel
	}
	target := reflect.ValueOf(destination)
	if !target.IsValid() || target.Kind() != reflect.Pointer || target.IsNil() || target.Elem().Type() != p.model {
		return ErrInvalidTarget
	}
	if len(sheet.Rows) == 0 {
		return ErrSheetEmpty
	}
	result := reflect.New(p.model).Elem()
	if err := p.decodePlans(ctx, sheet, p.blocks, result); err != nil {
		return err
	}
	if p.modelValidation != nil {
		if err := p.modelValidation(ctx, result.Addr().Interface()); err != nil {
			return locatedError(KindValidation, sheet.Name, "", "", -1, -1, err)
		}
	}
	target.Elem().Set(result)
	return nil
}

func (p *Plan) decodePlans(ctx context.Context, sheet Sheet, plans []blockPlan, result reflect.Value) error {
	consumed := make([]bool, len(sheet.Rows))
	for _, plan := range plans {
		if err := ctx.Err(); err != nil {
			return locatedError(KindCanceled, sheet.Name, plan.key, "", -1, -1, err)
		}
		blocks, err := plan.workflow.Select(ctx, sheet, plan.request)
		if err != nil {
			return locatedError(KindSplit, sheet.Name, plan.key, "", -1, -1, err)
		}
		if len(blocks) == 0 {
			if plan.request.Optional {
				continue
			}
			return locatedError(KindSplit, sheet.Name, plan.key, "", -1, -1, fmt.Errorf("workflow selected no blocks"))
		}
		previousEnd := -1
		selectedBlocks := make([]Block, 0, len(blocks))
		for _, block := range blocks {
			if err := validateBlock(sheet, block); err != nil {
				return locatedError(KindSplit, sheet.Name, plan.key, "", -1, -1, err)
			}
			if previousEnd > block.StartRow {
				return locatedError(KindSplit, sheet.Name, plan.key, "", sheet.absoluteRow(block.StartRow), 0, fmt.Errorf("%w: workflow blocks are overlapping or out of source order", ErrInvalidBlock))
			}
			previousEnd = block.EndRow
			consumedEnd := block.ConsumedEndRow
			if consumedEnd == 0 {
				consumedEnd = block.EndRow
			}
			for row := block.StartRow; row < consumedEnd; row++ {
				if consumed[row] {
					return locatedError(KindSplit, sheet.Name, plan.key, "", sheet.absoluteRow(row), 0, fmt.Errorf("row is consumed by multiple blocks"))
				}
				consumed[row] = true
			}
			if !plan.request.Optional || blockContainsMappedData(block, plan.request) {
				selectedBlocks = append(selectedBlocks, block)
			}
		}
		if len(selectedBlocks) == 0 {
			continue
		}
		value, err := p.decodeSelectedBlocks(ctx, sheet, plan, selectedBlocks)
		if err != nil {
			return err
		}
		if plan.format != FormatGroup {
			if err := validateMappingValue(ctx, value); err != nil {
				return locatedError(KindValidation, sheet.Name, plan.key, "", sheet.absoluteRow(selectedBlocks[0].StartRow), 0, err)
			}
		}
		field, err := writableFieldByIndex(result, plan.index)
		if err != nil {
			return locatedError(KindConversion, sheet.Name, plan.key, "", sheet.absoluteRow(selectedBlocks[0].StartRow), 0, err)
		}
		field.Set(value)
	}
	if p.unconsumed == RejectUnconsumedRows {
		for row, used := range consumed {
			if !used && !sheet.Rows[row].Empty() {
				return locatedError(KindUnconsumedRow, sheet.Name, "", "", sheet.absoluteRow(row), 0, fmt.Errorf("non-empty row was not consumed"))
			}
		}
	}
	if err := validateMappingValue(ctx, result); err != nil {
		return locatedError(KindValidation, sheet.Name, "", "", -1, -1, err)
	}
	return nil
}

func blockContainsMappedData(block Block, request BlockRequest) bool {
	return !rangeContainsNoData(block.Rows, 0, len(block.Rows), request)
}

func (p *Plan) decodeSelectedBlocks(ctx context.Context, sheet Sheet, plan blockPlan, blocks []Block) (reflect.Value, error) {
	if plan.format == FormatGroup {
		return p.decodeGroups(ctx, sheet, plan, blocks)
	}
	if len(blocks) > 1 && plan.format != FormatSlice {
		return reflect.Value{}, locatedError(KindSplit, sheet.Name, plan.key, "", sheet.absoluteRow(blocks[1].StartRow), 0, fmt.Errorf("workflow selected multiple blocks for %s", plan.format))
	}
	if plan.format == FormatSlice {
		result := reflect.MakeSlice(plan.typeOf, 0, 0)
		for _, block := range blocks {
			value, err := p.decodeBlock(ctx, sheet, plan, block)
			if err != nil {
				return reflect.Value{}, err
			}
			result = reflect.AppendSlice(result, value)
		}
		return result, nil
	}
	return p.decodeBlock(ctx, sheet, plan, blocks[0])
}

func (p *Plan) decodeGroups(ctx context.Context, sheet Sheet, plan blockPlan, blocks []Block) (reflect.Value, error) {
	if plan.typeOf.Kind() != reflect.Slice && len(blocks) > 1 {
		return reflect.Value{}, locatedError(KindSplit, sheet.Name, plan.key, "", sheet.absoluteRow(blocks[1].StartRow), 0, fmt.Errorf("repeated group requires a slice field"))
	}
	items := reflect.MakeSlice(reflect.SliceOf(plan.itemType), 0, len(blocks))
	for _, block := range blocks {
		item := reflect.New(plan.itemType).Elem()
		if err := p.decodePlans(ctx, sheet.scope(block), plan.children, item); err != nil {
			return reflect.Value{}, err
		}
		items = reflect.Append(items, item)
	}
	if plan.typeOf.Kind() == reflect.Slice {
		result := reflect.MakeSlice(plan.typeOf, 0, items.Len())
		for i := 0; i < items.Len(); i++ {
			item, err := convertGroupItem(items.Index(i), plan.typeOf.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			result = reflect.Append(result, item)
		}
		return result, nil
	}
	return convertGroupItem(items.Index(0), plan.typeOf)
}

func convertGroupItem(value reflect.Value, target reflect.Type) (reflect.Value, error) {
	if target.Kind() != reflect.Pointer {
		if value.Type().AssignableTo(target) {
			return value, nil
		}
		return reflect.Value{}, fmt.Errorf("group value %s is incompatible with %s", value.Type(), target)
	}
	pointer := reflect.New(target.Elem())
	converted, err := convertGroupItem(value, target.Elem())
	if err != nil {
		return reflect.Value{}, err
	}
	pointer.Elem().Set(converted)
	return pointer, nil
}

func writableFieldByIndex(value reflect.Value, index []int) (reflect.Value, error) {
	for position, fieldIndex := range index {
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				if !value.CanSet() {
					return reflect.Value{}, fmt.Errorf("field path is not settable")
				}
				value.Set(reflect.New(value.Type().Elem()))
			}
			value = value.Elem()
		}
		if value.Kind() != reflect.Struct || fieldIndex < 0 || fieldIndex >= value.NumField() {
			return reflect.Value{}, fmt.Errorf("invalid field path")
		}
		value = value.Field(fieldIndex)
		if position < len(index)-1 && value.Kind() != reflect.Struct && value.Kind() != reflect.Pointer {
			return reflect.Value{}, fmt.Errorf("invalid field path")
		}
	}
	if !value.CanSet() {
		return reflect.Value{}, fmt.Errorf("field path is not settable")
	}
	return value, nil
}

func (p *Plan) decodeBlock(ctx context.Context, sheet Sheet, plan blockPlan, block Block) (reflect.Value, error) {
	sheetName := sheet.Name
	if plan.blockCodec != nil {
		value, err := plan.blockCodec.Decode(ctx, BlockCodecContext{Sheet: sheetName, Block: plan.key, Options: plan.codecOptions}, block, plan.typeOf)
		if err != nil {
			return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, "", sheet.absoluteRow(block.StartRow), 0, err)
		}
		if !value.IsValid() || !value.Type().AssignableTo(plan.typeOf) {
			return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, "", sheet.absoluteRow(block.StartRow), 0, fmt.Errorf("block codec returned incompatible type"))
		}
		return value, nil
	}
	if plan.format == FormatForm {
		return p.decodeForm(ctx, sheetName, plan, block)
	}
	if plan.format == FormatTranspose {
		return p.decodeTranspose(ctx, sheetName, plan, block)
	}
	if plan.format == FormatSingle {
		valueRow := plan.request.DataRow
		if valueRow >= len(block.Rows) || len(block.Rows[valueRow]) == 0 {
			return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, "", sheet.absoluteRow(block.StartRow), 0, fmt.Errorf("single block is empty"))
		}
		if err := validateBlockLabel(p, plan, block.Rows[valueRow]); err != nil {
			return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, "", block.Rows[valueRow][0].Row, plan.request.LabelCol, err)
		}
		cells, err := blockValueCells(block.Rows[valueRow], plan.request.ValueCol, plan.multiCell, plan.separator)
		if err != nil {
			return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, "", block.Rows[valueRow][0].Row, plan.request.ValueCol, err)
		}
		if cellsEmpty(cells) && plan.allowEmpty {
			return reflect.Zero(plan.typeOf), nil
		}
		return plan.valueCodec.Decode(ctx, ValueCodecContext{Sheet: sheetName, Block: plan.key, Cell: cells[0], Options: plan.codecOptions}, cells, plan.typeOf)
	}
	headerRow := 0
	if plan.request.Title != "" {
		headerRow++
	}
	if headerRow >= len(block.Rows) {
		return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, "", sheet.absoluteRow(block.StartRow), 0, fmt.Errorf("header row is missing"))
	}
	headers := block.Rows[headerRow]
	columns := make(map[string]int, len(headers))
	for i, cell := range headers {
		key := p.normalizeTitle(cell.Value)
		if key != "" {
			if _, exists := columns[key]; exists {
				return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, "", cell.Row, cell.Col, fmt.Errorf("duplicate header %q", cell.Value))
			}
			columns[key] = i
		}
	}
	for _, field := range plan.fields {
		if _, exists := columns[p.normalizeTitle(field.header)]; !exists && field.required {
			row := sheet.absoluteRow(block.StartRow + headerRow)
			if len(headers) > 0 {
				row = headers[0].Row
			}
			return reflect.Value{}, locatedError(KindSplit, sheetName, plan.key, field.name, row, 0, fmt.Errorf("required header %q is missing", field.header))
		}
	}
	itemContainerType := reflect.SliceOf(plan.itemType)
	if plan.typeOf.Kind() == reflect.Slice {
		itemContainerType = plan.typeOf
	}
	items := reflect.MakeSlice(itemContainerType, 0, max(0, len(block.Rows)-headerRow-1))
	for rowIndex := headerRow + 1; rowIndex < len(block.Rows); rowIndex++ {
		row := block.Rows[rowIndex]
		if row.Empty() {
			continue
		}
		item := reflect.New(plan.itemType).Elem()
		for _, field := range plan.fields {
			if field.skipDecode {
				continue
			}
			column, exists := columns[p.normalizeTitle(field.header)]
			if !exists {
				continue
			}
			cells := []Cell{row[column]}
			if field.multiCell {
				cells = boundedFieldCells(row, column, columns, field.separator)
			}
			empty := true
			for _, cell := range cells {
				if strings.TrimSpace(cell.Value) != "" {
					empty = false
					break
				}
			}
			if empty {
				if field.required && !field.allowEmpty {
					return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, field.name, row[column].Row, column, fmt.Errorf("value is required"))
				}
				continue
			}
			value, err := field.valueCodec.Decode(ctx, ValueCodecContext{Sheet: sheetName, Block: plan.key, Field: field.name, Cell: row[column], Options: field.codecOptions}, cells, field.typeOf)
			if err != nil {
				return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, field.name, row[column].Row, column, err)
			}
			if !value.IsValid() || !value.Type().AssignableTo(field.typeOf) {
				return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, field.name, row[column].Row, column, fmt.Errorf("value codec returned incompatible type"))
			}
			if field.validate != "" && p.validation != nil {
				issue := ValidationIssue{Field: field.name, Rule: field.validate, Value: row[column].Value}
				if err := p.validation(issue); err != nil {
					return reflect.Value{}, locatedError(KindValidation, sheetName, plan.key, field.name, row[column].Row, column, err)
				}
			}
			item.FieldByIndex(field.index).Set(value)
		}
		if plan.typeOf.Kind() == reflect.Slice {
			converted, err := convertGroupItem(item, plan.typeOf.Elem())
			if err != nil {
				return reflect.Value{}, locatedError(KindConversion, sheetName, plan.key, "", row[0].Row, 0, err)
			}
			items = reflect.Append(items, converted)
		} else {
			items = reflect.Append(items, item)
		}
	}
	if plan.typeOf.Kind() == reflect.Slice {
		return items, nil
	}
	if items.Len() == 0 {
		return reflect.Zero(plan.typeOf), nil
	}
	return convertGroupItem(items.Index(0), plan.typeOf)
}

func (p *Plan) normalizeTitle(value string) string {
	if p.trimTitle {
		value = strings.TrimSpace(value)
	}
	if p.caseInsensitiveTitle {
		value = strings.ToLower(value)
	}
	return value
}
