// Package gormx provides GORM-backed Casbin enforcer helpers.
package gormx

import (
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/scoming-dev/tools/casbinx"
	"gorm.io/gorm"
)

var ErrNilDB = errors.New("casbinx/gormx: db is nil")

// NewEnforcer creates a Casbin enforcer backed by a GORM adapter.
func NewEnforcer(db *gorm.DB, m model.Model, prefix, tableName string) (*casbin.Enforcer, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if m == nil {
		m = casbinx.NewRBACModel()
	}

	adapter, err := gormadapter.NewAdapterByDBUseTableName(db, prefix, tableName)
	if err != nil {
		return nil, fmt.Errorf("casbinx/gormx: create adapter: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("casbinx/gormx: create enforcer: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("casbinx/gormx: load policy: %w", err)
	}
	return enforcer, nil
}

// InitRBACCasbin initializes and caches a classic RBAC enforcer.
func InitRBACCasbin(db *gorm.DB, prefix, tableName string) (*casbin.Enforcer, error) {
	if casbinx.Enforcer != nil {
		return casbinx.Enforcer, nil
	}
	enforcer, err := NewEnforcer(db, casbinx.NewRBACModel(), prefix, tableName)
	if err != nil {
		return nil, err
	}
	casbinx.Enforcer = enforcer
	return casbinx.Enforcer, nil
}

// InitDomainRBACCasbin initializes and caches a tenant/domain RBAC enforcer.
func InitDomainRBACCasbin(db *gorm.DB, prefix, tableName string) (*casbin.Enforcer, error) {
	if casbinx.Enforcer != nil {
		return casbinx.Enforcer, nil
	}
	enforcer, err := NewEnforcer(db, casbinx.NewDomainRBACModel(), prefix, tableName)
	if err != nil {
		return nil, err
	}
	casbinx.Enforcer = enforcer
	return casbinx.Enforcer, nil
}
