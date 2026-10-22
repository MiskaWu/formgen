package yaml

import gen "github.com/miskawu/formgen/internal/generator"

// New 建立一個新的 YAML Generator
func New(opts ...Option) gen.Generator {
	return newGenerator(opts...)
}
