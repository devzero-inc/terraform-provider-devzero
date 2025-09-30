package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestClusterResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Instantiate the resource and call Schema
	clusterResource := NewClusterResource()
	clusterResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	// Validate the schema
	validateClusterSchema(t, resp.Schema)
}

func validateClusterSchema(t *testing.T, schema schema.Schema) {
	// Validate required attributes
	requiredAttrs := []string{"name"}
	for _, attr := range requiredAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Required attribute %s not found in schema", attr)
		}
	}

	// Validate computed attributes
	computedAttrs := []string{"id", "token"}
	for _, attr := range computedAttrs {
		if attrSchema, exists := schema.Attributes[attr]; exists {
			if !attrSchema.IsComputed() {
				t.Errorf("Attribute %s should be computed", attr)
			}
		}
	}

	// Validate sensitive attributes
	if tokenAttr, exists := schema.Attributes["token"]; exists {
		if !tokenAttr.IsSensitive() {
			t.Error("token attribute should be sensitive")
		}
	}
}
