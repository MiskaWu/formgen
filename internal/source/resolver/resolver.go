package resolver

import "github.com/miskawu/formgen/internal/schema"

// Resolver 負責解析跨表引用，將 *entry 等參考類型展開為嵌套結構
type Resolver interface {
	// Resolve 將基礎 Entity 列表（含 *entry 等修飾符）
	// 展開為最終 Entity 列表（含嵌套類型定義）
	Resolve() ([]schema.Entity, error)
}

// New 創建新的 Resolver
func New(entities []schema.Entity, opts ...Option) Resolver {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// 建立 Entity 索引
	entityIndex := make(map[string]*schema.Entity)
	for i := range entities {
		entityIndex[entities[i].Name] = &entities[i]
	}

	return &engine{
		entities:    entities,
		entityIndex: entityIndex,
		opts:        o,
	}
}
