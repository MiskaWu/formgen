// Package strcase 提供字串命名格式轉換工具
package strcase

import (
	"strings"
)

// ToSnakeCase 將 camelCase/PascalCase 轉為 snake_case
// 保留原有底線，僅在小寫字母後跟大寫字母時插入底線
// 例如: userID -> user_id, Demo_SideQuest -> demo_side_quest, ID -> id
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		// 大寫字母前添加底線（僅當前一個字符是小寫字母時）
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(s[i-1])
			if prev >= 'a' && prev <= 'z' {
				result.WriteRune('_')
			}
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// ToPascalCase 將字串轉為 PascalCase
// 保留底線，僅將每個單字首字母大寫
// 例如: user_id -> User_Id, sideQuest -> SideQuest
// 若需移除底線，調用者可自行使用 strings.ReplaceAll(s, "_", "")
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '_' {
			capitalizeNext = true
			result.WriteRune(r) // 保留底線
			continue
		}
		if capitalizeNext && r >= 'a' && r <= 'z' {
			result.WriteRune(r - 32) // 轉大寫
		} else {
			result.WriteRune(r)
		}
		capitalizeNext = false
	}

	return result.String()
}
