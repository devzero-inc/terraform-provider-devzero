package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestClusterIDByNameDataSourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds := NewClusterIDByNameDataSource()
	ds.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	validateClusterIDByNameSchema(t, resp.Schema)
}

func validateClusterIDByNameSchema(t *testing.T, s schema.Schema) {
	requiredAttrs := []string{"name"}
	for _, attr := range requiredAttrs {
		a, exists := s.Attributes[attr]
		if !exists {
			t.Errorf("Required attribute %q not found in schema", attr)
			continue
		}
		if a.IsComputed() {
			t.Errorf("Attribute %q should not be computed", attr)
		}
	}

	optionalAttrs := []string{"team_id", "region", "cloud_provider", "liveness"}
	for _, attr := range optionalAttrs {
		if _, exists := s.Attributes[attr]; !exists {
			t.Errorf("Optional attribute %q not found in schema", attr)
		}
	}

	computedAttrs := []string{"cluster_id"}
	for _, attr := range computedAttrs {
		a, exists := s.Attributes[attr]
		if !exists {
			t.Errorf("Computed attribute %q not found in schema", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Attribute %q should be computed", attr)
		}
	}
}