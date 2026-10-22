package orm

import "strings"

// goReservedKeywords 是 Go 語言保留關鍵字和預留名稱列表
// 當欄位名與這些關鍵字衝突時，會自動加上後綴
var goReservedKeywords = map[string]bool{
	// Go 保留字
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,

	// GORM model 預留名稱（避免與方法衝突）
	"TableName": true,
	"ToProto":   true,

	// 常見的內建類型名稱（避免混淆）
	"string":  true,
	"int":     true,
	"int8":    true,
	"int16":   true,
	"int32":   true,
	"int64":   true,
	"uint":    true,
	"uint8":   true,
	"uint16":  true,
	"uint32":  true,
	"uint64":  true,
	"float32": true,
	"float64": true,
	"bool":    true,
	"byte":    true,
	"rune":    true,
	"error":   true,

	// 常見的內建函數名稱
	"append":  true,
	"cap":     true,
	"close":   true,
	"complex": true,
	"copy":    true,
	"delete":  true,
	"imag":    true,
	"len":     true,
	"make":    true,
	"new":     true,
	"panic":   true,
	"print":   true,
	"println": true,
	"real":    true,
	"recover": true,

	// 常見的內建常量
	"true":  true,
	"false": true,
	"iota":  true,
	"nil":   true,
}

// IsReservedKeyword 檢查名稱是否為 Go 保留關鍵字（大小寫敏感）
func IsReservedKeyword(name string) bool {
	// Go 是大小寫敏感的，但轉為 PascalCase 後首字母會大寫
	// 所以我們檢查原始名稱（小寫版本）
	return goReservedKeywords[strings.ToLower(name)]
}
