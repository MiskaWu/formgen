package sql

import (
	_ "embed"

	gen "github.com/miskawu/formgen/internal/generator"
)

//go:embed templates/sql.tmpl
var sqlTemplate string

// New 創建 sql 生成器，支援選項模式
func New(opts ...Option) gen.Generator {
	return newGenerator(opts...)
}
