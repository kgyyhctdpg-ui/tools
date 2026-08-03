package permissionx

import "testing"

func TestCodeParseAndMatch(t *testing.T) {
	code := Code("/users/*", "GET")
	if code != "/users/*:GET" {
		t.Fatalf("unexpected code: %q", code)
	}
	permission, err := Parse(code)
	if err != nil {
		t.Fatal(err)
	}
	if permission.Resource != "/users/*" || permission.Action != "GET" {
		t.Fatalf("unexpected permission: %#v", permission)
	}
	if !Match("/users/*:GET", "/users/1:GET") {
		t.Fatal("resource wildcard should match")
	}
	if !Match("/users/*:*", "/users/1:DELETE") {
		t.Fatal("action wildcard should match")
	}
	if Match("/users/*:GET", "/teams/1:GET") {
		t.Fatal("different resource should not match")
	}
}

func TestHasAndUnique(t *testing.T) {
	grants := Unique([]string{" user:create ", "user:create", "user:*", ""})
	if len(grants) != 2 {
		t.Fatalf("unexpected grants: %v", grants)
	}
	if !Has(grants, "user:delete") {
		t.Fatal("user:* should grant user:delete")
	}
	if !HasAny(grants, "team:create", "user:create") {
		t.Fatal("one permission should match")
	}
	if HasAll(grants, "user:create", "team:create") {
		t.Fatal("all permissions should not match")
	}
}
