package orm

// defaultMappings 定義標準化類型到 Go 類型的映射
// 輸入應為 YAML Schema 中的標準化類型（如 int32, int64, string）
var defaultMappings = map[string]string{
	// 整數類型
	"int8":   "int8",
	"int16":  "int16",
	"int32":  "int32",
	"int64":  "int64",
	"uint8":  "uint8",
	"uint16": "uint16",
	"uint32": "uint32",
	"uint64": "uint64",

	// 浮點數類型
	"float32": "float32",
	"float64": "float64",

	// 字符串和布爾
	"string": "string",
	"bool":   "bool",
	"bytes":  "[]byte",

	// 時間類型
	"timestamp": "time.Time",
	"datetime":  "time.Time",
	"date":      "time.Time",
	"time":      "time.Time",

	// 文字類型
	"text": "string",
	"json": "string",
}
