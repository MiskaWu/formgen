package generator

import "github.com/miskawu/formgen/internal/schema"

// Generator 定義代碼生成器的公開介面
// 所有生成器（SQL、ORM、Proto、YAML）都必須實現此介面
type Generator interface {
	Generate(entities []schema.Entity) ([]byte, error)
}
