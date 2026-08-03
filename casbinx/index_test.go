package casbinx

import (
	"testing"

	"github.com/casbin/casbin/v2"
)

func TestManagerRBAC(t *testing.T) {
	enforcer, err := casbin.NewEnforcer(NewRBACModel())
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(enforcer)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := manager.AddRoleForUser("alice", "admin"); err != nil || !ok {
		t.Fatalf("add role failed, ok=%v err=%v", ok, err)
	}
	if ok, err := manager.AddPermissionForRole("admin", "/users", "GET"); err != nil || !ok {
		t.Fatalf("add permission failed, ok=%v err=%v", ok, err)
	}
	ok, err := manager.Check("alice", "/users", "GET")
	if err != nil || !ok {
		t.Fatalf("expected permission, ok=%v err=%v", ok, err)
	}
	hasRole, err := manager.HasRoleForUser("alice", "admin")
	if err != nil || !hasRole {
		t.Fatalf("expected role, ok=%v err=%v", hasRole, err)
	}
	users, err := manager.GetUsersForRole("admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0] != "alice" {
		t.Fatalf("unexpected users: %v", users)
	}
	permissions, err := manager.GetPermissionsForRole("admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(permissions) != 1 || permissions[0][1] != "/users" || permissions[0][2] != "GET" {
		t.Fatalf("unexpected permissions: %v", permissions)
	}
	implicitPermissions, err := manager.GetImplicitPermissionsForUser("alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(implicitPermissions) == 0 {
		t.Fatal("expected implicit permissions")
	}
	results, err := manager.BatchCheck([]Request{
		{Subject: "alice", Object: "/users", Action: "GET"},
		{Subject: "alice", Object: "/users", Action: "DELETE"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0] || results[1] {
		t.Fatalf("unexpected batch results: %v", results)
	}
	if ok, err := manager.AddPermissionsForRole("editor", []string{"/articles", "POST"}, []string{"/articles", "PUT"}); err != nil || !ok {
		t.Fatalf("add permissions failed, ok=%v err=%v", ok, err)
	}
	if ok, err := manager.DeletePermissionsForRole("editor"); err != nil || !ok {
		t.Fatalf("delete permissions failed, ok=%v err=%v", ok, err)
	}
}

func TestManagerDomainRBAC(t *testing.T) {
	enforcer, err := casbin.NewEnforcer(NewDomainRBACModel())
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(enforcer)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := manager.AddRoleForUserInDomain("alice", "admin", "tenant-a"); err != nil || !ok {
		t.Fatalf("add domain role failed, ok=%v err=%v", ok, err)
	}
	if ok, err := manager.AddPermissionForRoleInDomain("admin", "tenant-a", "/users", "GET"); err != nil || !ok {
		t.Fatalf("add domain permission failed, ok=%v err=%v", ok, err)
	}
	if ok, err := manager.AddPermissionsForRoleInDomain("admin", "tenant-a", []string{"/users", "POST"}); err != nil || !ok {
		t.Fatalf("add domain permissions failed, ok=%v err=%v", ok, err)
	}
	ok, err := manager.CheckDomain("alice", "tenant-a", "/users", "GET")
	if err != nil || !ok {
		t.Fatalf("expected domain permission, ok=%v err=%v", ok, err)
	}
	ok, err = manager.CheckDomain("alice", "tenant-b", "/users", "GET")
	if err != nil || ok {
		t.Fatalf("tenant-b should not be allowed, ok=%v err=%v", ok, err)
	}
	roles, err := manager.GetRolesForUserInDomain("alice", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0] != "admin" {
		t.Fatalf("unexpected domain roles: %v", roles)
	}
	users, err := manager.GetUsersForRoleInDomain("admin", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0] != "alice" {
		t.Fatalf("unexpected domain users: %v", users)
	}
	permissions, err := manager.GetPermissionsForRoleInDomain("admin", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(permissions) != 2 {
		t.Fatalf("unexpected domain permissions: %v", permissions)
	}
	if ok, err := manager.DeletePermissionForRoleInDomain("admin", "tenant-a", "/users", "POST"); err != nil || !ok {
		t.Fatalf("delete domain permission failed, ok=%v err=%v", ok, err)
	}
	if ok, err := manager.DeleteRoleForUserInDomain("alice", "admin", "tenant-a"); err != nil || !ok {
		t.Fatalf("delete domain role failed, ok=%v err=%v", ok, err)
	}
}
