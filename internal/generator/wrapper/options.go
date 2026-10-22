package wrapper

import protogen "github.com/miskawu/formgen/internal/generator/proto"

// options 定義生成器的配置選項
type options struct {
	packageName       string // Go package name (default: "envelope")
	protoImportPath   string // Proto import path (e.g., "example.com/myapp/proto")
	protoPackageAlias string // Proto package alias (e.g., "pb")
	functionName      string // Function name (default: "wrapEnvelope")
	envelopeName      string // Envelope message name (default: "TableEnvelope")
}

// Option 定義選項函數類型
type Option func(*options)

// WithPackageName 設定生成的 Go 檔案的 package name
func WithPackageName(name string) Option {
	return func(o *options) {
		o.packageName = name
	}
}

// WithProtoPackage 設定 proto package 的 import 路徑和別名
// importPath: proto package 的完整 import 路徑，例如 "example.com/myapp/proto"
// alias: proto package 的別名，例如 "pb"。如果為空則不使用別名
func WithProtoPackage(importPath, alias string) Option {
	return func(o *options) {
		o.protoImportPath = importPath
		o.protoPackageAlias = alias
	}
}

// WithFunctionName 設定生成的函數名稱
func WithFunctionName(name string) Option {
	return func(o *options) {
		o.functionName = name
	}
}

// WithEnvelopeName 設定 envelope message 的名稱。
// 預設與 proto 生成器 WithEnvelope() 產生的 TableEnvelope 一致；
// 若使用自訂 plugin 模板定義了其他名稱的 envelope，可在此指定。
func WithEnvelopeName(name string) Option {
	return func(o *options) {
		o.envelopeName = name
	}
}

// defaultOptions 返回預設選項
func defaultOptions() *options {
	return &options{
		packageName:  "envelope",
		functionName: "wrapEnvelope",
		envelopeName: protogen.EnvelopeMessageName,
	}
}
