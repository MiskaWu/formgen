package proto

import "github.com/miskawu/formgen/internal/mapper"

// options 定義生成器的配置選項
type options struct {
	typeMapper        mapper.TypeMapper
	packageName       string // proto package 名稱
	goPackage         string // go_package 選項
	keywordSuffix     string // 關鍵字衝突時的後綴
	generateWrapper   bool   // 是否生成 wrapper message（用於快照序列化）
	generateEnvelope  bool   // 是否生成 envelope 結構（TableEnvelope、TableBundle）
	payloadFieldStart int    // oneof payload 的起始欄位編號
	pluginTemplate    string // plugin 模板路徑（"default" 使用內建模板）
}

// Option 定義選項函數類型
type Option func(*options)

// WithCustomMappings 使用自定義類型映射（會在預設映射基礎上添加/覆蓋）
func WithCustomMappings(mappings map[string]string) Option {
	return func(o *options) {
		m := mapper.NewTypeMapper(defaultMappings)
		m.AddMappings(mappings)
		o.typeMapper = m
	}
}

// WithPackageName 設定 proto package 名稱
func WithPackageName(name string) Option {
	return func(o *options) {
		o.packageName = name
	}
}

// WithGoPackage 設定 go_package 選項
func WithGoPackage(goPackage string) Option {
	return func(o *options) {
		o.goPackage = goPackage
	}
}

// WithKeywordSuffix 設定關鍵字衝突時的後綴（預設為 "_field"）
// 當欄位名與 Proto 保留關鍵字衝突時，會自動加上此後綴
func WithKeywordSuffix(suffix string) Option {
	return func(o *options) {
		o.keywordSuffix = suffix
	}
}

// WithWrapper 啟用 wrapper message 生成
// 為每個 message 生成對應的 List wrapper（如 DemoCategoryList）
// 用於快照序列化時包裝多筆數據
func WithWrapper() Option {
	return func(o *options) {
		o.generateWrapper = true
	}
}

// WithEnvelope 啟用 envelope 輸出格式（可選）
// 除了 Entity messages 與 List wrappers 之外，再生成：
//   - TableEnvelope：key、checksum 加上一個 oneof payload，涵蓋所有 List wrapper
//   - TableBundle：一組 TableEnvelope，方便一次傳送多張表
//
// 適合「用同一個 message 型別傳送任意一張表」的情境，搭配 wrapper 生成器產生的
// Go 包裝函數使用。會自動啟用 wrapper 生成。
func WithEnvelope() Option {
	return func(o *options) {
		o.generateWrapper = true
		o.generateEnvelope = true
	}
}

// WithPayloadFieldStart 設定 oneof payload 的起始欄位編號
// 預設為 10，讓 TableEnvelope 保留前 9 個編號給固定欄位
func WithPayloadFieldStart(start int) Option {
	return func(o *options) {
		o.payloadFieldStart = start
	}
}

// WithPluginTemplate 使用 plugin 模板生成
// path 可以是 "default"（使用內建模板）或外部 .tmpl 檔案路徑
// 會自動啟用 wrapper 生成
func WithPluginTemplate(path string) Option {
	return func(o *options) {
		o.pluginTemplate = path
		o.generateWrapper = true
	}
}

// defaultOptions 返回預設選項
func defaultOptions() *options {
	return &options{
		typeMapper:        mapper.NewTypeMapper(defaultMappings),
		packageName:       "schema",
		goPackage:         "",
		keywordSuffix:     "_field",
		payloadFieldStart: 10,
	}
}
