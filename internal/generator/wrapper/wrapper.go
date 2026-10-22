package wrapper

import (
	_ "embed"

	gen "github.com/miskawu/formgen/internal/generator"
)

//go:embed templates/wrapper.tmpl
var wrapperTemplate string

// New 創建 wrapper 生成器，支援選項模式
func New(opts ...Option) gen.Generator {
	return newGenerator(opts...)
}
