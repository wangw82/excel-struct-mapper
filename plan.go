package mapper

import (
	"context"
	"fmt"
	"reflect"
	"sort"
)

type fieldPlan struct {
	name         string
	header       string
	index        []int
	typeOf       reflect.Type
	required     bool
	allowEmpty   bool
	skipDecode   bool
	skipEncode   bool
	multiCell    bool
	separator    string
	codecOptions string
	valueCodec   ValueCodec
	validate     string
}

type blockPlan struct {
	key            string
	workflowName   WorkflowName
	workflow       BlockWorkflow
	blockCodec     BlockCodec
	blockCodecName BlockCodecName
	format         BlockFormat
	valueCodec     ValueCodec
	request        BlockRequest
	index          []int
	typeOf         reflect.Type
	itemType       reflect.Type
	fields         []fieldPlan
	children       []blockPlan
	multiCell      bool
	allowEmpty     bool
	separator      string
	codecOptions   string
}

type Plan struct {
	model                reflect.Type
	blocks               []blockPlan
	tags                 map[string][]tagPlan
	validation           func(ValidationIssue) error
	unconsumed           UnconsumedRowsPolicy
	trimTitle            bool
	caseInsensitiveTitle bool
	modelValidation      func(context.Context, any) error
}

func Compile[T any](options ...CompileOption) (*Plan, error) {
	return compile[T](nil, options)
}

func CompileWithBindings[T any](bindings []BlockBinding, options ...CompileOption) (*Plan, error) {
	return compile[T](bindings, options)
}

func compile[T any](bindings []BlockBinding, options []CompileOption) (*Plan, error) {
	typeOf := reflect.TypeOf((*T)(nil)).Elem()
	if typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}
	if typeOf.Kind() != reflect.Struct {
		return nil, ErrInvalidModel
	}
	config, err := newCompileConfig(options)
	if err != nil {
		return nil, err
	}
	plan := &Plan{
		model:                typeOf,
		tags:                 make(map[string][]tagPlan, len(config.registry.tagHandlers)),
		validation:           config.validation,
		unconsumed:           config.unconsumed,
		trimTitle:            config.trimTitle,
		caseInsensitiveTitle: config.caseInsensitiveTitle,
		modelValidation:      config.modelValidation,
	}
	keys := map[string]bool{}
	manual := make(map[string]BlockBinding, len(bindings))
	for _, block := range bindings {
		if block.Key == "" || block.Field == "" || block.Workflow == nil || block.Codec == nil {
			return nil, ErrInvalidBlockBinding
		}
		if _, exists := manual[block.Field]; exists {
			return nil, fmt.Errorf("%w: duplicate field %q", ErrInvalidBlockBinding, block.Field)
		}
		manual[block.Field] = block
	}
	if err := compileModelBlocks(typeOf, nil, "", config.registry, manual, keys, map[reflect.Type]bool{}, &plan.blocks); err != nil {
		return nil, err
	}
	for field := range manual {
		return nil, fmt.Errorf("%w: field %q does not exist", ErrInvalidBlockBinding, field)
	}
	if len(plan.blocks) == 0 {
		return nil, fmt.Errorf("%w: no blocks", ErrInvalidModel)
	}
	if err := validatePlanBoundaries(plan.blocks); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(config.registry.tagHandlers))
	for tag := range config.registry.tagHandlers {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		handlers := config.registry.tagHandlers[tag]
		tagPlans, err := compileTagPlans(typeOf, tag, handlers)
		if err != nil {
			return nil, err
		}
		plan.tags[tag] = tagPlans
	}
	return plan, nil
}
