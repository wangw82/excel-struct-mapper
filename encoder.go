package mapper

import (
	"context"
	"fmt"
	"reflect"
)

func (p *Plan) Encode(ctx context.Context, source any) (Sheet, error) {
	return p.encode(ctx, source)
}

func (p *Plan) encode(ctx context.Context, source any) (Sheet, error) {
	if p == nil {
		return Sheet{}, ErrInvalidModel
	}
	value := reflect.ValueOf(source)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return Sheet{}, ErrInvalidSource
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Type() != p.model {
		return Sheet{}, ErrInvalidSource
	}
	if err := validateMappingValue(ctx, value); err != nil {
		return Sheet{}, locatedError(KindValidation, "", "", "", -1, -1, err)
	}
	if p.modelValidation != nil {
		if err := p.modelValidation(ctx, addressableInterface(value)); err != nil {
			return Sheet{}, locatedError(KindValidation, "", "", "", -1, -1, err)
		}
	}
	output := sheetOutput{}
	if err := p.encodePlans(ctx, p.blocks, value, &output); err != nil {
		return Sheet{}, err
	}
	return output.Sheet(), nil
}

func addressableInterface(value reflect.Value) any {
	if value.CanAddr() {
		return value.Addr().Interface()
	}
	copyValue := reflect.New(value.Type())
	copyValue.Elem().Set(value)
	return copyValue.Interface()
}

func (p *Plan) encodePlans(ctx context.Context, plans []blockPlan, source reflect.Value, output *sheetOutput) error {
	for _, block := range plans {
		if err := ctx.Err(); err != nil {
			return locatedError(KindCanceled, "", block.key, "", -1, -1, err)
		}
		field, err := readableFieldByIndex(source, block.index)
		if err != nil {
			return locatedError(KindConversion, "", block.key, "", -1, -1, err)
		}
		if block.request.Optional && isEmptyMappedValue(field) {
			continue
		}
		if err := validateMappingValue(ctx, field); err != nil {
			return locatedError(KindValidation, "", block.key, "", -1, -1, err)
		}
		var blocks [][]Line
		if block.format == FormatGroup {
			blocks, err = p.encodeGroups(ctx, block, field)
		} else {
			var lines []Line
			lines, err = p.encodeBlock(ctx, block, field)
			blocks = [][]Line{lines}
		}
		if err != nil {
			return err
		}
		request := block.request
		if block.format == FormatGroup && isBuiltinWorkflow(block.workflowName) {
			request.MinRows = 0
		}
		if err := block.workflow.Place(ctx, output, request, blocks); err != nil {
			return locatedError(KindConfiguration, "", block.key, "", -1, -1, err)
		}
	}
	return nil
}

func isBuiltinWorkflow(name WorkflowName) bool {
	switch name {
	case WorkflowAll, WorkflowIndex, WorkflowStart, WorkflowTitle, WorkflowRepeatTitle, WorkflowTitleRange:
		return true
	default:
		return false
	}
}

func (p *Plan) encodeGroups(ctx context.Context, plan blockPlan, value reflect.Value) ([][]Line, error) {
	items := value
	if value.Kind() != reflect.Slice {
		items = reflect.MakeSlice(reflect.SliceOf(value.Type()), 1, 1)
		items.Index(0).Set(value)
	}
	if items.Len() == 0 {
		return nil, locatedError(KindConversion, "", plan.key, "", -1, -1, fmt.Errorf("%w: group slice is empty", ErrInvalidSource))
	}
	blocks := make([][]Line, 0, items.Len())
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i)
		for item.Kind() == reflect.Pointer {
			if item.IsNil() {
				return nil, locatedError(KindConversion, "", plan.key, "", -1, -1, ErrInvalidSource)
			}
			item = item.Elem()
		}
		output := sheetOutput{}
		if err := p.encodePlans(ctx, plan.children, item, &output); err != nil {
			return nil, err
		}
		if len(output.Lines) < plan.request.MinRows {
			return nil, locatedError(KindConfiguration, "", plan.key, "", -1, -1, fmt.Errorf("%w: group has %d rows, minimum is %d", ErrLayoutConflict, len(output.Lines), plan.request.MinRows))
		}
		blocks = append(blocks, output.Lines)
	}
	return blocks, nil
}

func readableFieldByIndex(value reflect.Value, index []int) (reflect.Value, error) {
	for _, fieldIndex := range index {
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return reflect.Value{}, ErrInvalidSource
			}
			value = value.Elem()
		}
		if value.Kind() != reflect.Struct || fieldIndex < 0 || fieldIndex >= value.NumField() {
			return reflect.Value{}, ErrInvalidSource
		}
		value = value.Field(fieldIndex)
	}
	return value, nil
}

func (p *Plan) encodeBlock(ctx context.Context, plan blockPlan, value reflect.Value) ([]Line, error) {
	if plan.blockCodec != nil {
		lines, err := plan.blockCodec.Encode(ctx, BlockCodecContext{Block: plan.key, Options: plan.codecOptions}, value)
		if err != nil {
			return nil, locatedError(KindConversion, "", plan.key, "", -1, -1, err)
		}
		return lines, nil
	}
	if plan.format == FormatForm {
		return p.encodeForm(ctx, plan, value)
	}
	if plan.format == FormatTranspose {
		return p.encodeTranspose(ctx, plan, value)
	}
	if plan.format == FormatSingle {
		cells, err := plan.valueCodec.Encode(ctx, ValueCodecContext{Block: plan.key, Options: plan.codecOptions}, value)
		if err != nil {
			return nil, err
		}
		if !plan.multiCell && len(cells) > 1 {
			return nil, fmt.Errorf("block %s requires multi_cell=true", plan.key)
		}
		line := make(Line, max(plan.request.LabelCol+1, plan.request.ValueCol+len(cells)))
		if plan.request.Label != "" {
			line[plan.request.LabelCol] = plan.request.Label
		}
		copy(line[plan.request.ValueCol:], cells)
		if plan.separator != "" {
			line = append(line, plan.separator)
		}
		lines := make([]Line, contentDataRow(plan))
		return append(lines, line), nil
	}
	items := value
	if value.Kind() != reflect.Slice {
		items = reflect.MakeSlice(reflect.SliceOf(value.Type()), 1, 1)
		items.Index(0).Set(value)
	}
	widths := make([]int, len(plan.fields))
	for i := range widths {
		widths[i] = 1
	}
	encoded := make([][][]string, items.Len())
	for row := 0; row < items.Len(); row++ {
		item := items.Index(row)
		for item.Kind() == reflect.Pointer {
			if item.IsNil() {
				return nil, ErrInvalidSource
			}
			item = item.Elem()
		}
		encoded[row] = make([][]string, len(plan.fields))
		for column, field := range plan.fields {
			if field.skipEncode {
				continue
			}
			values, err := field.valueCodec.Encode(ctx, ValueCodecContext{Block: plan.key, Field: field.name, Options: field.codecOptions}, item.FieldByIndex(field.index))
			if err != nil {
				return nil, locatedError(KindConversion, "", plan.key, field.name, row, column, err)
			}
			if !field.multiCell && len(values) > 1 {
				return nil, fmt.Errorf("field %s requires multi_cell=true", field.name)
			}
			if len(values) == 0 {
				values = []string{""}
			}
			encoded[row][column] = values
			fieldWidth := len(values)
			if field.separator != "" {
				fieldWidth++
			}
			if field.multiCell && fieldWidth > widths[column] {
				widths[column] = fieldWidth
			}
		}
	}
	var lines []Line
	header := Line{}
	for i, field := range plan.fields {
		if field.skipEncode {
			continue
		}
		header = append(header, field.header)
		for extra := 1; extra < widths[i]; extra++ {
			header = append(header, "")
		}
	}
	lines = append(lines, header)
	for _, item := range encoded {
		line := Line{}
		for i, field := range plan.fields {
			if field.skipEncode {
				continue
			}
			values := append([]string(nil), item[i]...)
			if field.separator != "" {
				values = append(values, field.separator)
			}
			line = append(line, values...)
			for len(values) < widths[i] {
				line = append(line, "")
				values = append(values, "")
			}
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func isEmptyMappedValue(value reflect.Value) bool {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return true
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		return value.IsNil() || value.Len() == 0
	default:
		return value.IsZero()
	}
}
