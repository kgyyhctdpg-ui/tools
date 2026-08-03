package casbinx

import (
	"errors"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

var (
	Enforcer *casbin.Enforcer
)

var ErrNilEnforcer = errors.New("casbinx: enforcer is nil")

// 清空缓存
func Close() {
	Enforcer = nil
}

// NewRBACModel creates a classic sub/obj/act RBAC model.
func NewRBACModel() model.Model {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act")
	return m
}

// NewDomainRBACModel creates an RBAC model with tenant/domain isolation.
func NewDomainRBACModel() model.Model {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, dom, obj, act")
	m.AddDef("p", "p", "sub, dom, obj, act")
	m.AddDef("g", "g", "_, _, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act")
	return m
}

// Request is a single permission check.
type Request struct {
	Subject string
	Object  string
	Action  string
}

// DomainRequest is a permission check scoped to a domain or tenant.
type DomainRequest struct {
	Subject string
	Domain  string
	Object  string
	Action  string
}

// Manager wraps common Casbin RBAC operations.
type Manager struct {
	Enforcer *casbin.Enforcer
}

func NewManager(enforcer *casbin.Enforcer) (*Manager, error) {
	if enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return &Manager{Enforcer: enforcer}, nil
}

func (manager *Manager) Check(subject, object, action string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.Enforce(subject, object, action)
}

func (manager *Manager) BatchCheck(requests []Request) ([]bool, error) {
	result := make([]bool, 0, len(requests))
	for _, request := range requests {
		ok, err := manager.Check(request.Subject, request.Object, request.Action)
		if err != nil {
			return nil, err
		}
		result = append(result, ok)
	}
	return result, nil
}

func (manager *Manager) CheckDomain(subject, domain, object, action string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.Enforce(subject, domain, object, action)
}

func (manager *Manager) BatchCheckDomain(requests []DomainRequest) ([]bool, error) {
	result := make([]bool, 0, len(requests))
	for _, request := range requests {
		ok, err := manager.CheckDomain(request.Subject, request.Domain, request.Object, request.Action)
		if err != nil {
			return nil, err
		}
		result = append(result, ok)
	}
	return result, nil
}

func (manager *Manager) AddRoleForUser(user, role string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.AddRoleForUser(user, role)
}

func (manager *Manager) DeleteRoleForUser(user, role string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeleteRoleForUser(user, role)
}

func (manager *Manager) GetRolesForUser(user string) ([]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetRolesForUser(user)
}

func (manager *Manager) AddPermissionForRole(role, object, action string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.AddPermissionForUser(role, object, action)
}

func (manager *Manager) AddPermissionsForRole(role string, permissions ...[]string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.AddPermissionsForUser(role, permissions...)
}

func (manager *Manager) DeletePermissionForRole(role, object, action string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeletePermissionForUser(role, object, action)
}

func (manager *Manager) DeletePermissionsForRole(role string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeletePermissionsForUser(role)
}

func (manager *Manager) GetPermissionsForRole(role string) ([][]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetPermissionsForUser(role)
}

func (manager *Manager) GetImplicitPermissionsForUser(user string) ([][]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetImplicitPermissionsForUser(user)
}

func (manager *Manager) HasRoleForUser(user, role string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.HasRoleForUser(user, role)
}

func (manager *Manager) DeleteRolesForUser(user string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeleteRolesForUser(user)
}

func (manager *Manager) DeleteUser(user string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeleteUser(user)
}

func (manager *Manager) DeleteRole(role string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeleteRole(role)
}

func (manager *Manager) GetUsersForRole(role string) ([]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetUsersForRole(role)
}

func (manager *Manager) GetImplicitRolesForUser(user string) ([]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetImplicitRolesForUser(user)
}

func (manager *Manager) LoadPolicy() error {
	if manager == nil || manager.Enforcer == nil {
		return ErrNilEnforcer
	}
	return manager.Enforcer.LoadPolicy()
}

func (manager *Manager) SavePolicy() error {
	if manager == nil || manager.Enforcer == nil {
		return ErrNilEnforcer
	}
	return manager.Enforcer.SavePolicy()
}

func (manager *Manager) AddRoleForUserInDomain(user, role, domain string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.AddRoleForUserInDomain(user, role, domain)
}

func (manager *Manager) AddPermissionForRoleInDomain(role, domain, object, action string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.AddPolicy(role, domain, object, action)
}

func (manager *Manager) AddPermissionsForRoleInDomain(role, domain string, permissions ...[]string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	rules := make([][]string, 0, len(permissions))
	for _, permission := range permissions {
		rule := append([]string{role, domain}, permission...)
		rules = append(rules, rule)
	}
	return manager.Enforcer.AddPolicies(rules)
}

func (manager *Manager) DeleteRoleForUserInDomain(user, role, domain string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeleteRoleForUserInDomain(user, role, domain)
}

func (manager *Manager) DeleteRolesForUserInDomain(user, domain string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeleteRolesForUserInDomain(user, domain)
}

func (manager *Manager) GetRolesForUserInDomain(user, domain string) ([]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetRolesForUserInDomain(user, domain), nil
}

func (manager *Manager) GetUsersForRoleInDomain(role, domain string) ([]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetUsersForRoleInDomain(role, domain), nil
}

func (manager *Manager) DeletePermissionForRoleInDomain(role, domain, object, action string) (bool, error) {
	if manager == nil || manager.Enforcer == nil {
		return false, ErrNilEnforcer
	}
	return manager.Enforcer.DeletePermissionForUser(role, domain, object, action)
}

func (manager *Manager) GetPermissionsForRoleInDomain(role, domain string) ([][]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetPermissionsForUser(role, domain)
}

func (manager *Manager) GetImplicitPermissionsForUserInDomain(user, domain string) ([][]string, error) {
	if manager == nil || manager.Enforcer == nil {
		return nil, ErrNilEnforcer
	}
	return manager.Enforcer.GetImplicitPermissionsForUser(user, domain)
}
