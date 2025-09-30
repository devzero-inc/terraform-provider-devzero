package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetStringList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test with valid string list
	values := []attr.Value{
		types.StringValue("value1"),
		types.StringValue("value2"),
		types.StringValue("value3"),
	}

	result, err := getStringList(ctx, values)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"value1", "value2", "value3"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}

	for i, expectedValue := range expected {
		if result[i] != expectedValue {
			t.Errorf("Expected %s at index %d, got %s", expectedValue, i, result[i])
		}
	}
}

func TestGetStringMap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test with valid string map
	values := map[string]attr.Value{
		"key1": types.StringValue("value1"),
		"key2": types.StringValue("value2"),
		"key3": types.StringValue("value3"),
	}

	result, err := getStringMap(ctx, values)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}

	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("Expected %s for key %s, got %s", expectedValue, key, result[key])
		}
	}
}

func TestGetElementList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test with valid element list
	values := []attr.Value{
		types.StringValue("item1"),
		types.StringValue("item2"),
		types.StringValue("item3"),
	}

	// Test function that converts string to uppercase
	convertFunc := func(ctx context.Context, value string) (string, error) {
		return value + "_UPPERCASE", nil
	}

	result, err := getElementList(ctx, values, convertFunc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"item1_UPPERCASE", "item2_UPPERCASE", "item3_UPPERCASE"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}

	for i, expectedValue := range expected {
		if result[i] != expectedValue {
			t.Errorf("Expected %s at index %d, got %s", expectedValue, i, result[i])
		}
	}
}

func TestFromStringList(t *testing.T) {
	t.Parallel()

	// Test with valid string slice
	input := []string{"value1", "value2", "value3"}

	result := fromStringList(input)

	expected := []attr.Value{
		types.StringValue("value1"),
		types.StringValue("value2"),
		types.StringValue("value3"),
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}

	for i, expectedValue := range expected {
		if result[i].(types.String).ValueString() != expectedValue.(types.String).ValueString() {
			t.Errorf("Expected %s at index %d, got %s", expectedValue.(types.String).ValueString(), i, result[i].(types.String).ValueString())
		}
	}
}

func TestFromStringMap(t *testing.T) {
	t.Parallel()

	// Test with valid string map
	input := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	result := fromStringMap(input)

	expected := map[string]attr.Value{
		"key1": types.StringValue("value1"),
		"key2": types.StringValue("value2"),
		"key3": types.StringValue("value3"),
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}

	for key, expectedValue := range expected {
		if result[key].(types.String).ValueString() != expectedValue.(types.String).ValueString() {
			t.Errorf("Expected %s for key %s, got %s", expectedValue.(types.String).ValueString(), key, result[key].(types.String).ValueString())
		}
	}
}

func TestFromElementList(t *testing.T) {
	t.Parallel()

	// Test with valid element list
	input := []string{"item1", "item2", "item3"}

	// Create a simple attr types map for the test
	attrTypes := map[string]attr.Type{
		"value": types.StringType,
	}

	result := fromElementList(input, attrTypes)

	// The function should return a list of ObjectValue items
	if len(result) != len(input) {
		t.Errorf("Expected %d items, got %d", len(input), len(result))
	}

	// Each result should be an ObjectValue
	for i, item := range result {
		if item == nil {
			t.Errorf("Expected non-nil item at index %d", i)
		}
	}
}
