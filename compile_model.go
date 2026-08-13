package mapper

import (
	"fmt"
	"reflect"
	"strings"
)

func compileModelBlocks(typeOf reflect.Type, parentIndex []int, prefix string, registry *Registry, manual map[string]BlockBinding, keys map[string]bool, stack map[reflect.Type]bool, blocks *[]blockPlan) error {
	if stack[typeOf] {
		return fmt.Errorf("%w: recursive anonymous embedding at %q", ErrInvalidModel, prefix)
	}
	stack[typeOf] = true
	defer delete(stack, typeOf)
	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		path := field.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		index := append(append([]int(nil), parentIndex...), field.Index...)
		tag, tagged := field.Tag.Lookup(tagNameExcel)
		custom, customOK := manual[path]
		if customOK && tagged && tag != "-" {
			return fmt.Errorf("%w: field %q also has an excel tag", ErrInvalidBlockBinding, path)
		}
		if customOK {
			if !field.IsExported() {
				return fmt.Errorf("%w: field %q is not exported", ErrInvalidBlockBinding, path)
			}
			block := blockPlan{key: custom.Key, workflow: custom.Workflow, blockCodec: custom.Codec, request: custom.Request, index: index, typeOf: field.Type}
			if keys[block.key] {
				return fmt.Errorf("%w: %q", ErrDuplicateKey, block.key)
			}
			keys[block.key] = true
			*blocks = append(*blocks, block)
			delete(manual, path)
			continue
		}
		if tagged && tag != "-" {
			if !field.IsExported() {
				return fmt.Errorf("field %s: %w: mapped field is not exported", path, ErrInvalidModel)
			}
			options, err := parseOptions(tag, blockOptions)
			if err != nil {
				return fmt.Errorf("field %s: %w", path, err)
			}
			block, err := compileBlock(field, options, registry)
			if err != nil {
				return err
			}
			block.index = index
			if block.format == FormatGroup {
				block.request.unframed = true
				if block.workflowName == WorkflowTitleRange {
					block.request.includeEndBlock = true
				}
				if err := compileModelBlocks(block.itemType, nil, path, registry, manual, map[string]bool{}, stack, &block.children); err != nil {
					return err
				}
				if len(block.children) == 0 {
					return fmt.Errorf("field %s: %w: group has no mapped blocks", path, ErrInvalidModel)
				}
				if err := validateGroupPlan(path, block); err != nil {
					return err
				}
			}
			if keys[block.key] {
				return fmt.Errorf("%w: %q", ErrDuplicateKey, block.key)
			}
			keys[block.key] = true
			*blocks = append(*blocks, block)
			continue
		}
		nested := field.Type
		for nested.Kind() == reflect.Pointer {
			nested = nested.Elem()
		}
		if nested.Kind() == reflect.Struct && (field.Anonymous || hasManualDescendant(manual, path)) {
			if err := compileModelBlocks(nested, index, path, registry, manual, keys, stack, blocks); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateGroupPlan(path string, block blockPlan) error {
	workflow := block.workflowName
	if block.typeOf.Kind() == reflect.Slice && workflow != WorkflowRepeatTitle {
		if workflow == WorkflowAll || workflow == WorkflowIndex || workflow == WorkflowStart || workflow == WorkflowTitleRange {
			return fmt.Errorf("field %s: %w: a group slice requires workflow %q", path, ErrInvalidTag, WorkflowRepeatTitle)
		}
	}
	if block.typeOf.Kind() != reflect.Slice && workflow == WorkflowRepeatTitle {
		return fmt.Errorf("field %s: %w: workflow %q requires a group slice", path, ErrInvalidTag, WorkflowRepeatTitle)
	}
	if workflow == WorkflowRepeatTitle && block.children[0].request.Title != block.request.Title {
		return fmt.Errorf("field %s: %w: repeated group title must match its first child block title", path, ErrInvalidTag)
	}
	if workflow == WorkflowRepeatTitle && block.request.BlankLine {
		return fmt.Errorf("field %s: %w: repeated groups use the next matching title as their boundary", path, ErrInvalidTag)
	}
	if workflow == WorkflowTitleRange && block.request.EndTitle != "" && block.children[len(block.children)-1].request.Title != block.request.EndTitle {
		return fmt.Errorf("field %s: %w: range end title must match its last child block title", path, ErrInvalidTag)
	}
	if workflow == WorkflowTitleRange && block.request.EndTitle != "" && !block.children[len(block.children)-1].request.BlankLine {
		return fmt.Errorf("field %s: %w: range end child must use blank_line=true", path, ErrInvalidTag)
	}
	if workflow == WorkflowTitleRange && block.request.Title != "" && block.children[0].request.Title != block.request.Title {
		return fmt.Errorf("field %s: %w: range start title must match its first child block title", path, ErrInvalidTag)
	}
	if workflow == WorkflowTitle && block.children[0].request.Title != block.request.Title {
		return fmt.Errorf("field %s: %w: group title must match its first child block title", path, ErrInvalidTag)
	}
	return nil
}

func validatePlanBoundaries(blocks []blockPlan) error {
	for i, block := range blocks {
		if err := validatePlanBoundaries(block.children); err != nil {
			return err
		}
		if block.workflowName != WorkflowRepeatTitle || block.request.EndTitle == "" {
			continue
		}
		if i+1 >= len(blocks) || blocks[i+1].request.Title != block.request.EndTitle {
			return fmt.Errorf("block %q: %w: repeat_title end_title must match the next sibling block title", block.key, ErrInvalidTag)
		}
	}
	return nil
}

func hasManualDescendant(manual map[string]BlockBinding, path string) bool {
	prefix := path + "."
	for field := range manual {
		if strings.HasPrefix(field, prefix) {
			return true
		}
	}
	return false
}
