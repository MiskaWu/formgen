package proto

import "strings"

var (
	// defaultMappings 是 Proto 類型的預設映射
	defaultMappings = map[string]string{
		// 整數類型
		"int":      "int32",
		"int8":     "int32",
		"int16":    "int32",
		"int32":    "int32",
		"int64":    "int64",
		"uint":     "uint32",
		"uint8":    "uint32",
		"uint16":   "uint32",
		"uint32":   "uint32",
		"uint64":   "uint64",
		"tinyint":  "int32",
		"shortint": "int32",

		// 浮點數類型
		"float":   "float",
		"float32": "float",
		"float64": "double",
		"double":  "double",

		// 字符串和布爾
		"string": "string",
		"bool":   "bool",
		"byte":   "bytes",
		"bytes":  "bytes",

		// 時間類型（使用 google.protobuf.Timestamp 或 int64）
		"timestamp": "int64",
		"datetime":  "int64",
		"date":      "string",
		"time":      "string",

		// 文字類型
		"text": "string",
		"json": "string",

		// 特殊類型
		"entry": "int32",  // 參考 ID
		"tag":   "string", // 標籤
		"asset": "string", // 資源路徑
	}

	// protoReservedKeywords 是 Protocol Buffers 保留關鍵字列表
	// 當欄位名與這些關鍵字衝突時，會自動加上後綴
	protoReservedKeywords = map[string]bool{
		// Proto 語法關鍵字
		"syntax":     true,
		"import":     true,
		"weak":       true,
		"public":     true,
		"package":    true,
		"option":     true,
		"message":    true,
		"service":    true,
		"rpc":        true,
		"returns":    true,
		"stream":     true,
		"enum":       true,
		"extend":     true,
		"extensions": true,
		"reserved":   true,
		"to":         true,
		"max":        true,
		"oneof":      true,
		"map":        true,
		"group":      true,

		// 欄位修飾符
		"required": true,
		"optional": true,
		"repeated": true,
		"singular": true,

		// 內建類型（避免混淆）
		"double":   true,
		"float":    true,
		"int32":    true,
		"int64":    true,
		"uint32":   true,
		"uint64":   true,
		"sint32":   true,
		"sint64":   true,
		"fixed32":  true,
		"fixed64":  true,
		"sfixed32": true,
		"sfixed64": true,
		"bool":     true,
		"string":   true,
		"bytes":    true,

		// 常量
		"true":  true,
		"false": true,
		"inf":   true,
		"nan":   true,
	}
)

// IsReservedKeyword 檢查名稱是否為 Proto 保留關鍵字（大小寫敏感）
func IsReservedKeyword(name string) bool {
	return protoReservedKeywords[strings.ToLower(name)]
}
