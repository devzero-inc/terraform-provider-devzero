package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// valueAs decodes a framework attr.Value into target using the framework's
// reflection rules (tfsdk.ValueAs). Unlike tftypes.Value.As, this supports
// tfsdk-tagged structs (e.g. Taint, RawKarpenterSpec) as well as primitives, so
// it can be used generically for list/map element conversion.
func valueAs(ctx context.Context, value attr.Value, target any) error {
	diags := tfsdk.ValueAs(ctx, value, target)
	if !diags.HasError() {
		return nil
	}
	var errs []error
	for _, d := range diags.Errors() {
		errs = append(errs, fmt.Errorf("%s: %s", d.Summary(), d.Detail()))
	}
	return errors.Join(errs...)
}

func getElementList[T any, V any](ctx context.Context, values []attr.Value, converter func(ctx context.Context, value V) (T, error)) ([]T, error) {
	var elements []T
	var errs []error
	for _, value := range values {
		var v V
		if err := valueAs(ctx, value, &v); err != nil {
			errs = append(errs, err)
			continue
		}

		element, err := converter(ctx, v)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		elements = append(elements, element)

	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return elements, nil
}

func getStringList(ctx context.Context, values []attr.Value) ([]string, error) {
	return getElementList(ctx, values, func(ctx context.Context, value string) (string, error) {
		return value, nil
	})
}

func getElementMap[K comparable, V any, T any](ctx context.Context, values map[K]attr.Value, converter func(ctx context.Context, value V) (T, error)) (map[K]T, error) {
	elements := make(map[K]T)
	var errs []error
	for key, value := range values {
		var v V
		if err := valueAs(ctx, value, &v); err != nil {
			errs = append(errs, err)
			continue
		}
		element, err := converter(ctx, v)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		elements[key] = element
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return elements, nil
}

func getStringMap(ctx context.Context, values map[string]attr.Value) (map[string]string, error) {
	return getElementMap(ctx, values, func(ctx context.Context, value string) (string, error) {
		return value, nil
	})
}

func fromElementList[T any](values []T, attrTypes map[string]attr.Type) []attr.Value {
	var elementList []attr.Value
	for _, value := range values {
		element, _ := types.ObjectValueFrom(context.Background(), attrTypes, value)
		elementList = append(elementList, element)
	}
	return elementList
}

func fromStringList(values []string) []attr.Value {
	var stringList []attr.Value
	for _, value := range values {
		stringList = append(stringList, types.StringValue(value))
	}
	return stringList
}

func fromStringMap(values map[string]string) map[string]attr.Value {
	stringMap := make(map[string]attr.Value)
	for key, value := range values {
		stringMap[key] = types.StringValue(value)
	}
	return stringMap
}
