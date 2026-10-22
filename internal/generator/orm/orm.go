package orm

import (
	_ "embed"

	gen "github.com/miskawu/formgen/internal/generator"
)

//go:embed templates/orm.tmpl
var ormTemplate string

// New 創建 ORM 生成器，支援選項模式
func New(opts ...Option) gen.Generator {
	return newGenerator(opts...)
}
