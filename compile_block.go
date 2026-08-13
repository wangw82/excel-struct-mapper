package mapper

import (
	"fmt"
	"reflect"
)

func compileBlock(field reflect.StructField, options tagOptions, registry *Registry) (blockPlan, error) {
	workflowName := WorkflowName(options[tagKeyWorkflow])
	block := blockPlan{
		key: options[tagKeyKey], workflowName: workflowName,
		format: BlockFormat(options[tagKeyFormat]), typeOf: field.Type,
		blockCodecName: BlockCodecName(options[tagKeyBlockCodec]),
		codecOptions:   options[tagKeyCodecOptions],
	}
	if block.key == "" {
		return block, fmt.Errorf("field %s: %w: key is required", field.Name, ErrInvalidTag)
	}
	if workflowName == "" {
		workflowName = WorkflowAll
		block.workflowName = workflowName
	}
	if _, hasStart := options[tagKeyStartRow]; hasStart && workflowName != WorkflowIndex && workflowName != WorkflowStart && workflowName != WorkflowRepeatTitle {
		return block, fmt.Errorf("%w: start_row is not supported by workflow %q", ErrInvalidTag, workflowName)
	}
	if _, hasEnd := options[tagKeyEndRow]; hasEnd && workflowName != WorkflowIndex {
		return block, fmt.Errorf("%w: end_row is only supported by workflow %q", ErrInvalidTag, WorkflowIndex)
	}
	if _, hasEndTitle := options[tagKeyEndTitle]; hasEndTitle && workflowName != WorkflowTitleRange && workflowName != WorkflowRepeatTitle {
		return block, fmt.Errorf("%w: end_title is only supported by workflow %q or %q", ErrInvalidTag, WorkflowTitleRange, WorkflowRepeatTitle)
	}
	if block.format == "" {
		block.format = inferBlockFormat(field.Type)
	}
	if _, exists := options[tagKeyValueCodec]; exists && block.format != FormatSingle {
		return block, fmt.Errorf("%w: block value_codec requires format %q", ErrInvalidTag, FormatSingle)
	}
	if block.blockCodecName != "" {
		if _, exists := options[tagKeyValueCodec]; exists {
			return block, fmt.Errorf("%w: block_codec cannot be combined with value_codec", ErrInvalidTag)
		}
	}
	if _, exists := options[tagKeyLabel]; exists && block.format != FormatSingle {
		return block, fmt.Errorf("%w: label requires format %q", ErrInvalidTag, FormatSingle)
	}
	if _, exists := options[tagKeySeparator]; exists && block.format != FormatSingle {
		return block, fmt.Errorf("%w: block separator requires format %q", ErrInvalidTag, FormatSingle)
	}
	if _, exists := options[tagKeyMultiCell]; exists && block.format != FormatSingle {
		return block, fmt.Errorf("%w: block multi_cell requires format %q", ErrInvalidTag, FormatSingle)
	}
	if _, exists := options[tagKeyAllowEmpty]; exists && block.format != FormatSingle {
		return block, fmt.Errorf("%w: block allow_empty requires format %q", ErrInvalidTag, FormatSingle)
	}
	if workflowName == WorkflowRepeatTitle && block.format != FormatSlice && block.format != FormatGroup {
		return block, fmt.Errorf("%w: workflow %q requires format %q or %q", ErrInvalidTag, workflowName, FormatSlice, FormatGroup)
	}
	block.workflow = registry.workflows[workflowName]
	if block.workflow == nil {
		return block, fmt.Errorf("%w: %q", ErrUnknownWorkflow, workflowName)
	}
	if block.format != FormatStruct && block.format != FormatSlice && block.format != FormatSingle && block.format != FormatGroup && block.format != FormatForm && block.format != FormatTranspose {
		return block, fmt.Errorf("%w: %q", ErrUnknownBlockFormat, block.format)
	}
	if block.blockCodecName != "" {
		if block.format == FormatGroup {
			return block, fmt.Errorf("%w: block_codec cannot be combined with format %q", ErrInvalidTag, FormatGroup)
		}
		block.blockCodec = registry.blockCodecs[block.blockCodecName]
		if block.blockCodec == nil {
			return block, fmt.Errorf("%w: %q", ErrUnknownBlockCodec, block.blockCodecName)
		}
	}
	var err error
	block.request.StartRow, err = optionRow(options, tagKeyStartRow, 0)
	if err != nil {
		return block, err
	}
	block.request.EndRow, err = optionRow(options, tagKeyEndRow, block.request.StartRow)
	if err != nil {
		return block, err
	}
	if value, ok := options[tagKeyMinRows]; ok {
		block.request.MinRows, err = parsePositive(value)
		if err != nil {
			return block, err
		}
	}
	block.request.BlankLine, err = optionBool(options, tagKeyBlankLine, false)
	if err != nil {
		return block, err
	}
	block.request.Optional, err = optionBool(options, tagKeyOptional, false)
	if err != nil {
		return block, err
	}
	block.request.includeEndBlock, err = optionBool(options, tagKeyIncludeEndBlock, false)
	if err != nil {
		return block, err
	}
	if block.request.includeEndBlock && workflowName != WorkflowTitleRange {
		return block, fmt.Errorf("%w: include_end_block requires workflow %q", ErrInvalidTag, WorkflowTitleRange)
	}
	if block.request.includeEndBlock && block.format != FormatGroup && block.blockCodec == nil {
		return block, fmt.Errorf("%w: include_end_block requires format %q or block_codec", ErrInvalidTag, FormatGroup)
	}
	if block.request.includeEndBlock && block.blockCodec != nil {
		block.request.unframed = true
	}
	block.allowEmpty, err = optionBool(options, tagKeyAllowEmpty, false)
	if err != nil {
		return block, err
	}
	block.multiCell, err = optionBool(options, tagKeyMultiCell, false)
	if err != nil {
		return block, err
	}
	block.separator = options[tagKeySeparator]
	if block.separator != "" && !block.multiCell {
		return block, fmt.Errorf("%w: separator requires multi_cell=true", ErrInvalidTag)
	}
	block.request.Title, block.request.EndTitle = options[tagKeyTitle], options[tagKeyEndTitle]
	defaultDataRow := 0
	if block.request.Title != "" {
		defaultDataRow = 1
	}
	block.request.DataRow, err = optionRow(options, tagKeyDataRow, defaultDataRow)
	if err != nil {
		return block, err
	}
	block.request.LabelCol, err = optionColumn(options, tagKeyLabelCol, 0)
	if err != nil {
		return block, err
	}
	block.request.Label = options[tagKeyLabel]
	defaultValueCol := 1
	if block.format == FormatSingle && block.request.Label == "" {
		defaultValueCol = 0
	}
	block.request.ValueCol, err = optionColumn(options, tagKeyValueCol, defaultValueCol)
	if err != nil {
		return block, err
	}
	if (workflowName == WorkflowTitle || workflowName == WorkflowRepeatTitle) && block.request.Title == "" {
		return block, fmt.Errorf("%w: title is required", ErrInvalidTag)
	}
	if workflowName == WorkflowTitleRange && block.request.Title == "" && block.request.EndTitle == "" {
		return block, fmt.Errorf("%w: title_range requires title or end_title", ErrInvalidTag)
	}
	item := field.Type
	if item.Kind() == reflect.Slice {
		item = item.Elem()
	}
	for item.Kind() == reflect.Pointer {
		item = item.Elem()
	}
	block.itemType = item
	if block.blockCodec != nil {
		return block, nil
	}
	if block.format == FormatSingle {
		valueCodecName := ValueCodecName(options[tagKeyValueCodec])
		if valueCodecName == "" {
			valueCodecName = ValueCodecBuiltin
		}
		block.valueCodec = registry.valueCodecs[valueCodecName]
		if block.valueCodec == nil {
			return block, fmt.Errorf("%w: %q", ErrUnknownValueCodec, valueCodecName)
		}
		return block, nil
	}
	if block.format == FormatGroup {
		if item.Kind() != reflect.Struct {
			return block, fmt.Errorf("field %s: %w: format %s requires struct values", field.Name, ErrInvalidModel, block.format)
		}
		return block, nil
	}
	if block.format == FormatForm {
		if field.Type.Kind() == reflect.Slice || item.Kind() != reflect.Struct {
			return block, fmt.Errorf("field %s: %w: format %s requires one struct", field.Name, ErrInvalidModel, block.format)
		}
	}
	if block.format == FormatTranspose && field.Type.Kind() != reflect.Slice {
		return block, fmt.Errorf("field %s: %w: format %s requires a slice", field.Name, ErrInvalidModel, block.format)
	}
	if block.format == FormatSlice && field.Type.Kind() != reflect.Slice {
		return block, fmt.Errorf("field %s: %w: format %s requires a slice", field.Name, ErrInvalidModel, block.format)
	}
	if block.format == FormatStruct && field.Type.Kind() == reflect.Slice {
		return block, fmt.Errorf("field %s: %w: format %s requires one struct", field.Name, ErrInvalidModel, block.format)
	}
	if item.Kind() != reflect.Struct {
		return block, fmt.Errorf("field %s: %w: format %s requires struct values", field.Name, ErrInvalidModel, block.format)
	}
	for i := 0; i < item.NumField(); i++ {
		child := item.Field(i)
		tag, ok := child.Tag.Lookup(tagNameExcel)
		if !ok || tag == "-" {
			continue
		}
		fieldPlan, err := compileField(child, tag, registry)
		if err != nil {
			return block, err
		}
		block.fields = append(block.fields, fieldPlan)
	}
	if len(block.fields) == 0 {
		return block, fmt.Errorf("field %s: no mapped columns", field.Name)
	}
	if block.format == FormatTranspose {
		for _, field := range block.fields {
			if field.multiCell {
				return block, fmt.Errorf("field %s: %w: transpose fields cannot use multi_cell", field.name, ErrInvalidTag)
			}
		}
	}
	return block, nil
}

func compileField(field reflect.StructField, tag string, registry *Registry) (fieldPlan, error) {
	if !field.IsExported() {
		return fieldPlan{}, fmt.Errorf("field %s: %w: mapped field is not exported", field.Name, ErrInvalidModel)
	}
	options, err := parseOptions(tag, fieldOptions)
	if err != nil {
		return fieldPlan{}, fmt.Errorf("field %s: %w", field.Name, err)
	}
	valueCodecName := ValueCodecName(options[tagKeyValueCodec])
	if valueCodecName == "" {
		valueCodecName = ValueCodecBuiltin
	}
	result := fieldPlan{name: field.Name, header: options[tagKeyHeader], index: field.Index, typeOf: field.Type, valueCodec: registry.valueCodecs[valueCodecName], validate: options[tagKeyValidate], separator: options[tagKeySeparator], codecOptions: options[tagKeyCodecOptions]}
	if result.header == "" {
		return result, fmt.Errorf("field %s: %w: header is required", field.Name, ErrInvalidTag)
	}
	result.required, err = optionBool(options, tagKeyRequired, false)
	if err != nil {
		return result, err
	}
	result.allowEmpty, err = optionBool(options, tagKeyAllowEmpty, false)
	if err != nil {
		return result, err
	}
	result.skipDecode, err = optionBool(options, tagKeySkipDecode, false)
	if err != nil {
		return result, err
	}
	result.skipEncode, err = optionBool(options, tagKeySkipEncode, false)
	if err != nil {
		return result, err
	}
	result.multiCell, err = optionBool(options, tagKeyMultiCell, false)
	if err != nil {
		return result, err
	}
	if result.separator != "" && !result.multiCell {
		return result, fmt.Errorf("field %s: %w: separator requires multi_cell=true", field.Name, ErrInvalidTag)
	}
	if result.required && result.allowEmpty {
		return result, fmt.Errorf("field %s: required and allow_empty cannot both be true", field.Name)
	}
	if result.valueCodec == nil {
		return result, fmt.Errorf("%w: %q", ErrUnknownValueCodec, valueCodecName)
	}
	return result, nil
}

func inferBlockFormat(value reflect.Type) BlockFormat {
	if value.Kind() == reflect.Slice {
		return FormatSlice
	}
	base := value
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if base.Kind() == reflect.Struct {
		return FormatStruct
	}
	return FormatSingle
}

func parsePositive(value string) (int, error) {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil || result < 0 {
		return 0, fmt.Errorf("%w: invalid non-negative integer %q", ErrInvalidTag, value)
	}
	return result, nil
}
