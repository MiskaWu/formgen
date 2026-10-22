package sql

// defaultMappings 定義標準化類型到 SQL 類型的映射
// 輸入應為 YAML Schema 中的標準化類型（如 int32, int64, string）
var defaultMappings = map[string]string{
	// 整數類型
	"int8":   "TINYINT",
	"int16":  "SMALLINT",
	"int32":  "INT",
	"int64":  "BIGINT",
	"uint8":  "TINYINT UNSIGNED",
	"uint16": "SMALLINT UNSIGNED",
	"uint32": "INT UNSIGNED",
	"uint64": "BIGINT UNSIGNED",

	// 浮點數類型
	"float32": "FLOAT",
	"float64": "DOUBLE",

	// 字符串和布爾
	"string": "VARCHAR(255)",
	"bool":   "BOOLEAN",
	"bytes":  "BLOB",

	// 時間類型
	"timestamp": "TIMESTAMP",
	"datetime":  "DATETIME",
	"date":      "DATE",
	"time":      "TIME",

	// 文字類型
	"text": "TEXT",
	"json": "JSON",
}
