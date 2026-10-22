package proto

import gen "github.com/miskawu/formgen/internal/generator"

// New 創建 proto 生成器，支援選項模式
func New(opts ...Option) gen.Generator {
	return newGenerator(opts...)
}
