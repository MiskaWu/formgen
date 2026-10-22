package yaml

import "github.com/miskawu/formgen/internal/mapper"

type options struct {
	typeMapper mapper.TypeMapper
}

// Option 是配置 YAML Generator 的函數類型
type Option func(*options)

func defaultOptions() *options {
	// 預設為空映射，必須由外部透過 WithMappings 注入
	return &options{
		typeMapper: mapper.NewTypeMapper(nil),
	}
}

// WithMappings 設定類型映射
// 這是必要的配置，通常從 config.GetCustomMappings() 取得
//
// 範例:
//
//	gen := New(WithMappings(config.GetCustomMappings()))
func WithMappings(mappings map[string]string) Option {
	return func(o *options) {
		o.typeMapper = mapper.NewTypeMapper(mappings)
	}
}
