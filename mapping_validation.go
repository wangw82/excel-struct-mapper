package mapper

import (
	"context"
	"reflect"
)

func validateMappingValue(ctx context.Context, value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil
	}
	if value.CanAddr() && value.Addr().CanInterface() {
		if validator, ok := value.Addr().Interface().(MappingValidator); ok {
			return validator.ValidateMapping(ctx)
		}
	}
	if value.CanInterface() {
		if validator, ok := value.Interface().(MappingValidator); ok {
			return validator.ValidateMapping(ctx)
		}
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		for i := 0; i < value.Len(); i++ {
			if err := validateMappingValue(ctx, value.Index(i)); err != nil {
				return err
			}
		}
	}
	if value.Kind() != reflect.Pointer && value.CanInterface() {
		copyValue := reflect.New(value.Type())
		copyValue.Elem().Set(value)
		if validator, ok := copyValue.Interface().(MappingValidator); ok {
			return validator.ValidateMapping(ctx)
		}
	}
	return nil
}
