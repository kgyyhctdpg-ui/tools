package gormx

import (
	"errors"
	"testing"

	"github.com/scoming-dev/tools/casbinx"
)

func TestNewEnforcerNilDB(t *testing.T) {
	casbinx.Close()
	_, err := NewEnforcer(nil, casbinx.NewRBACModel(), "", "casbin_rule")
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("expected ErrNilDB, got %v", err)
	}
}
