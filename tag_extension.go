package mapper

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// TagContext describes one application-owned struct tag invocation. Params is
// the unmodified text after the first equals sign. Children contains results
// produced by tagged fields nested below Field.
type TagContext struct {
	Tag      string
	Name     string
	Params   string
	Path     string
	Field    reflect.StructField
	Children []TagResult
}

// TagResult is one executed tag node. Output is owned by the registered
// handler; the mapper does not inspect or convert it.
type TagResult struct {
	Path     string
	Name     string
	Params   string
	Output   any
	Children []TagResult
}

// TagHandler implements one named behavior within an application-owned struct
// tag namespace.
type TagHandler interface {
	Handle(context.Context, TagContext) (any, error)
}

type TagHandlerFunc func(context.Context, TagContext) (any, error)

func (f TagHandlerFunc) Handle(ctx context.Context, tag TagContext) (any, error) {
	if f == nil {
		return nil, fmt.Errorf("tag handler is not configured")
	}
	return f(ctx, tag)
}

type tagPlan struct {
	path     string
	field    reflect.StructField
	name     string
	params   string
	handler  TagHandler
	children []tagPlan
}

// RunTag executes a registered application tag in model declaration order.
// Untagged intermediate structs are transparent and only contribute children.
func (p *Plan) RunTag(ctx context.Context, tag string) ([]TagResult, error) {
	if p == nil {
		return nil, ErrInvalidModel
	}
	plans, exists := p.tags[tag]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTag, tag)
	}
	return runTagPlans(ctx, tag, plans)
}

func runTagPlans(ctx context.Context, tag string, plans []tagPlan) ([]TagResult, error) {
	results := make([]TagResult, 0, len(plans))
	for _, plan := range plans {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		children, err := runTagPlans(ctx, tag, plan.children)
		if err != nil {
			return nil, err
		}
		if plan.handler == nil {
			results = append(results, children...)
			continue
		}
		context := TagContext{
			Tag: tag, Name: plan.name, Params: plan.params,
			Path: plan.path, Field: plan.field, Children: children,
		}
		output, err := plan.handler.Handle(ctx, context)
		if err != nil {
			return nil, fmt.Errorf("field %s tag %q handler %q: %w", plan.path, tag, plan.name, err)
		}
		results = append(results, TagResult{
			Path: plan.path, Name: plan.name, Params: plan.params,
			Output: output, Children: children,
		})
	}
	return results, nil
}

func compileTagPlans(typeOf reflect.Type, tag string, handlers map[string]TagHandler) ([]tagPlan, error) {
	return compileTagChildren(indirectTagType(typeOf), tag, handlers, "", map[reflect.Type]bool{})
}

func compileTagChildren(typeOf reflect.Type, tag string, handlers map[string]TagHandler, prefix string, stack map[reflect.Type]bool) ([]tagPlan, error) {
	if typeOf.Kind() != reflect.Struct || stack[typeOf] {
		return nil, nil
	}
	stack[typeOf] = true
	defer delete(stack, typeOf)

	plans := make([]tagPlan, 0)
	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		path := field.Name
		if prefix != "" {
			path = prefix + "." + field.Name
		}
		raw, tagged := field.Tag.Lookup(tag)
		if tagged && raw == "-" {
			continue
		}
		if tagged && !field.IsExported() {
			return nil, fmt.Errorf("field %s: %w: tagged field is not exported", path, ErrInvalidModel)
		}
		children, err := compileTagChildren(indirectTagType(field.Type), tag, handlers, path, stack)
		if err != nil {
			return nil, err
		}
		plan := tagPlan{path: path, field: field, children: children}
		if tagged {
			plan.name, plan.params, err = parseExtensionTag(raw)
			if err != nil {
				return nil, fmt.Errorf("field %s tag %q: %w", path, tag, err)
			}
			plan.handler = handlers[plan.name]
			if plan.handler == nil {
				return nil, fmt.Errorf("field %s tag %q: %w: %q", path, tag, ErrUnknownTagHandler, plan.name)
			}
		}
		if tagged || len(children) > 0 {
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func indirectTagType(typeOf reflect.Type) reflect.Type {
	for typeOf.Kind() == reflect.Pointer || typeOf.Kind() == reflect.Slice || typeOf.Kind() == reflect.Array {
		typeOf = typeOf.Elem()
	}
	return typeOf
}

func parseExtensionTag(value string) (name, params string, err error) {
	name, params, found := strings.Cut(value, "=")
	if name == "" || strings.TrimSpace(name) != name {
		return "", "", fmt.Errorf("%w: handler name is required", ErrInvalidTag)
	}
	if !found {
		return name, "", nil
	}
	return name, params, nil
}

func validateExtensionTagName(tag string) error {
	if tag == "" || tag == tagNameExcel || invalidTagToken(tag) {
		return fmt.Errorf("%w: invalid application tag %q", ErrInvalidRegistry, tag)
	}
	return nil
}

func validateTagHandlerName(name string) error {
	if name == "" || invalidTagToken(name) || strings.Contains(name, "=") || name == "-" {
		return fmt.Errorf("%w: invalid tag handler name %q", ErrInvalidRegistry, name)
	}
	return nil
}

func invalidTagToken(value string) bool {
	return strings.ContainsAny(value, "\"`\\:") || strings.ContainsFunc(value, func(char rune) bool {
		return unicode.IsSpace(char) || unicode.IsControl(char)
	})
}
