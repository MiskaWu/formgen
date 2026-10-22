package sql

import "github.com/miskawu/formgen/internal/mapper"

// options 定義生成器的配置選項
type options struct {
	typeMapper    mapper.TypeMapper
	keywordSuffix string // 關鍵字衝突時的後綴
}

// Option 定義選項函數類型
type Option func(*options)

// WithCustomMappings 使用自定義類型映射（會在預設映射基礎上添加/覆蓋）
func WithCustomMappings(mappings map[string]string) Option {
	return func(o *options) {
		mapper := mapper.NewTypeMapper(defaultMappings)
		mapper.AddMappings(mappings)
		o.typeMapper = mapper
	}
}

// WithKeywordSuffix 設定關鍵字衝突時的後綴（預設為 "_field"）
// 當欄位名與 SQL 保留關鍵字衝突時，會自動加上此後綴
func WithKeywordSuffix(suffix string) Option {
	return func(o *options) {
		o.keywordSuffix = suffix
	}
}

// defaultOptions 返回預設選項
func defaultOptions() *options {
	return &options{
		typeMapper:    mapper.NewTypeMapper(defaultMappings),
		keywordSuffix: "_field",
	}
}
