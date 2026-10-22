package resolver

import "github.com/miskawu/formgen/internal/schema"

// FindEmbeddedEntities 分析已解析的 entities，找出被嵌入的 Entity 名稱集合。
//
// 被嵌入的條件：
//   - 有其他 Entity 的欄位引用它（IsReference=true）
//   - 該引用是整個嵌入（RefField=""，表示嵌套整個 Entity）
//   - NestedColumns 有內容（表示解析成功，不是 fallback 的情況）
//
// 這些 Entity 不需要單獨寫入資料庫，因為它們會跟著嵌入它的對象一起序列化。
//
// 注意：此函數應在 Resolver.Resolve() 之後調用，因為需要 NestedColumns 資訊。
func FindEmbeddedEntities(entities []schema.Entity) map[string]bool {
	embedded := make(map[string]bool)

	for _, entity := range entities {
		for _, col := range entity.Columns {
			// 檢查是否為有效的嵌入引用
			// 1. IsReference=true：是參考類型
			// 2. RefEntity != ""：有參考目標
			// 3. RefField == ""：嵌入整個 Entity（不是引用特定欄位）
			// 4. len(NestedColumns) > 0：解析成功（不是 fallback 的情況）
			if col.IsReference &&
				col.RefEntity != "" &&
				col.RefField == "" &&
				len(col.NestedColumns) > 0 {
				embedded[col.RefEntity] = true
			}
		}
	}

	return embedded
}

// FindReferencedEntities 分析已解析的 entities，找出所有被引用的 Entity 名稱集合。
// 這包含：
//   - 被完整嵌入的 Entity（RefField=="" 且有 NestedColumns）
//   - 被字段引用的 Entity（RefField!=""）
//
// 這些 Entity 的數據需要在導入時被索引，以便查找嵌入數據。
//
// 注意：完整嵌入需要 NestedColumns（由 Resolver 填充），字段引用不需要。
func FindReferencedEntities(entities []schema.Entity) map[string]bool {
	referenced := make(map[string]bool)

	for _, entity := range entities {
		for _, col := range entity.Columns {
			if !col.IsReference || col.RefEntity == "" {
				continue
			}

			// 完整嵌入：RefField=="" 且有 NestedColumns
			// 字段引用：RefField!=""（不需要 NestedColumns）
			if (col.RefField == "" && len(col.NestedColumns) > 0) || col.RefField != "" {
				referenced[col.RefEntity] = true
			}
		}
	}

	return referenced
}
