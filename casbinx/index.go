package casbinx

import (
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	Enforcer *casbin.Enforcer
)

// 初始化 casbin
func InitRBACCasbin(db *gorm.DB) (*casbin.Enforcer, error) {

	// 判断是否有缓存
	if Enforcer != nil {
		return Enforcer, nil
	}

	adapter, err := gormadapter.NewAdapterByDBUseTableName(db, "system", "casbin")
	if err != nil {
		log.Printf("连接数据库错误: %v", err)
		return nil, err
	}

	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act")

	Enforcer, err = casbin.NewEnforcer(m, adapter)
	if err != nil {
		log.Printf("初始化casbin错误: %v", err)
		return nil, err
	}

	if err = Enforcer.LoadPolicy(); err != nil {
		log.Printf("加载策略失败: %v", err)
		return nil, err
	}

	return Enforcer, nil
}

// 清空缓存
func Close() {
	Enforcer = nil
}
