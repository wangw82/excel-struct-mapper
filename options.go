package mapper

import (
	"context"
	"fmt"
)

type compileConfig struct {
	registry             *Registry
	validation           func(ValidationIssue) error
	unconsumed           UnconsumedRowsPolicy
	trimTitle            bool
	caseInsensitiveTitle bool
	modelValidation      func(context.Context, any) error
}

func newCompileConfig(options []CompileOption) (*compileConfig, error) {
	config := &compileConfig{
		registry:             defaultRegistry(),
		trimTitle:            true,
		caseInsensitiveTitle: true,
	}
	for _, option := range options {
		if option != nil {
			if err := option.apply(config); err != nil {
				return nil, err
			}
		}
	}
	return config, nil
}

type UnconsumedRowsPolicy int

const (
	RejectUnconsumedRows UnconsumedRowsPolicy = iota
	IgnoreUnconsumedRows
)

type CompileOption interface {
	apply(*compileConfig) error
}

type compileOptionFunc func(*compileConfig) error

func (f compileOptionFunc) apply(config *compileConfig) error { return f(config) }

func WithRegistry(registry *Registry) CompileOption {
	return compileOptionFunc(func(config *compileConfig) error {
		if registry == nil {
			return ErrInvalidRegistry
		}
		config.registry = registry.Build()
		return nil
	})
}

func WithValidation(validation func(ValidationIssue) error) CompileOption {
	return compileOptionFunc(func(config *compileConfig) error { config.validation = validation; return nil })
}

// WithModelValidation validates the fully mapped model after decoding and
// before encoding. Nested values can implement MappingValidator for local rules.
func WithModelValidation(validation func(context.Context, any) error) CompileOption {
	return compileOptionFunc(func(config *compileConfig) error {
		config.modelValidation = validation
		return nil
	})
}

func WithUnconsumedRowsPolicy(policy UnconsumedRowsPolicy) CompileOption {
	return compileOptionFunc(func(config *compileConfig) error {
		if policy != RejectUnconsumedRows && policy != IgnoreUnconsumedRows {
			return fmt.Errorf("invalid unconsumed rows policy %d", policy)
		}
		config.unconsumed = policy
		return nil
	})
}

func WithTrimTitle(enabled bool) CompileOption {
	return compileOptionFunc(func(config *compileConfig) error { config.trimTitle = enabled; return nil })
}
func WithCaseInsensitiveTitle(enabled bool) CompileOption {
	return compileOptionFunc(func(config *compileConfig) error { config.caseInsensitiveTitle = enabled; return nil })
}
