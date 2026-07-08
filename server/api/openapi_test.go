package api

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestOpenAPIContractShape(t *testing.T) {
	doc := NewOpenAPI().OpenAPI()
	schemas := doc.Components.Schemas.Map()

	assertArrayRef(t, schemas["AdminMembers"].Properties["users"], "#/components/schemas/AdminUser")
	assertArrayRef(t, schemas["AdminMembers"].Properties["groups"], "#/components/schemas/AdminGroup")
	assertArrayRef(t, schemas["AdminJudgers"].Properties["judgers"], "#/components/schemas/AdminJudger")
	assertArrayRef(t, schemas["BackupList"].Properties["items"], "#/components/schemas/BackupItem")

	create := schemas["AdminJudgerCreate"]
	if _, ok := create.Properties["auth"]; ok {
		t.Fatal("AdminJudgerCreate must not expose auth")
	}
	if !required(create, "name") {
		t.Fatal("AdminJudgerCreate.name must be required")
	}

	update := schemas["AdminJudgerUpdate"]
	if required(update, "auth") {
		t.Fatal("AdminJudgerUpdate.auth must be optional")
	}

	if _, ok := doc.Paths["/api/problems/{id}.zip"]; !ok {
		t.Fatal("expected problem zip path")
	}
	if _, ok := doc.Paths["/api/problems/{id}/zip"]; ok {
		t.Fatal("unexpected internal problem zip path")
	}
}

func assertArrayRef(t *testing.T, schema *huma.Schema, ref string) {
	t.Helper()
	if schema == nil {
		t.Fatalf("missing schema for %s", ref)
	}
	if schema.Type != huma.TypeArray {
		t.Fatalf("expected array for %s, got %q", ref, schema.Type)
	}
	if schema.Nullable {
		t.Fatalf("expected non-null array for %s", ref)
	}
	if schema.Items == nil || schema.Items.Ref != ref {
		t.Fatalf("expected array item ref %s, got %#v", ref, schema.Items)
	}
}

func required(schema *huma.Schema, name string) bool {
	for _, item := range schema.Required {
		if item == name {
			return true
		}
	}
	return false
}
