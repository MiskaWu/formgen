package orm

import "github.com/miskawu/formgen/internal/mapper"

// options 定義生成器的配置選項
type options struct {
	typeMapper        mapper.TypeMapper
	packageName       string
	protoImportPath   string
	protoPackageAlias string
	keywordSuffix     string // 關鍵字衝突時的後綴
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

// WithPackageName 設定生成的 Go 檔案的 package name
func WithPackageName(name string) Option {
	return func(o *options) {
		o.packageName = name
	}
}

// WithProtoPackage 設定 proto package 的 import 路徑和別名，用於生成 ToProto 方法
// importPath: proto package 的完整 import 路徑，例如 "github.com/yourproject/proto"
// alias: proto package 的別名，例如 "pb"。如果為空則不使用別名
func WithProtoPackage(importPath, alias string) Option {
	return func(o *options) {
		o.protoImportPath = importPath
		o.protoPackageAlias = alias
	}
}

// WithKeywordSuffix 設定關鍵字衝突時的後綴（預設為 "_"）
// 當欄位名與 Go 保留關鍵字衝突時，會自動加上此後綴
func WithKeywordSuffix(suffix string) Option {
	return func(o *options) {
		o.keywordSuffix = suffix
	}
}

// defaultOptions 返回預設選項
func defaultOptions() *options {
	return &options{
		typeMapper:    mapper.NewTypeMapper(defaultMappings),
		packageName:   "models",
		keywordSuffix: "_",
	}
}
