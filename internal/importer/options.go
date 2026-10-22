package importer

import (
	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/serializer"
	"github.com/miskawu/formgen/internal/source"
)

// Option 用於配置 Importer
type Option func(*Importer)

// SnapshotConfig 快照配置
type SnapshotConfig struct {
	ProtoFilePath    string          // Proto 文件路徑
	Namespace        string          // 命名空間
	EmbeddedEntities map[string]bool // 被嵌入的 Entity 名稱集合（這些不會單獨生成 snapshot）
	Entities         []schema.Entity // Schema 資訊（用於識別嵌入欄位）
}

// WithBatchSize 設置批次處理大小
func WithBatchSize(size int) Option {
	return func(s *Importer) {
		s.batchSize = size
	}
}

// WithProgressCallback 設置進度回調函數
func WithProgressCallback(cb ProgressCallback) Option {
	return func(s *Importer) {
		s.onProgress = cb
	}
}

// WithSnapshot 啟用快照功能
// 將所有快照相關配置整合在一起，確保配置完整
//
// EmbeddedEntities 用於指定被嵌入的 Entity 名稱集合。
// 這些 Entity 的數據仍會寫入資料庫（DB 表是扁平的），
// 但不會單獨生成 snapshot（因為數據已嵌入到父 Entity 的 protobuf 中）。
//
// Entities 用於識別嵌入欄位，序列化時會查找被引用表的完整數據。
func WithSnapshot(config SnapshotConfig) Option {
	return func(s *Importer) {
		ser, err := serializer.NewProtoSerializer(config.ProtoFilePath)
		if err != nil {
			// 如果創建失敗，不啟用快照
			return
		}
		s.protoSerializer = ser
		s.enableSnapshot = true
		s.namespace = config.Namespace
		s.embeddedEntities = config.EmbeddedEntities

		// 建立 Entity schema 索引
		if len(config.Entities) > 0 {
			s.entityIndex = buildEntityIndex(config.Entities)
			s.dataIndex = make(map[string]*tableData)
		}
	}
}

// buildEntityIndex 從 entities 建立 schema 索引
func buildEntityIndex(entities []schema.Entity) map[string]*entitySchema {
	index := make(map[string]*entitySchema)

	for _, entity := range entities {
		es := &entitySchema{
			name:    entity.Name,
			columns: make(map[string]*columnSchema),
		}

		for _, col := range entity.Columns {
			cs := &columnSchema{
				name: col.Name,
			}

			if col.IsReference && col.RefEntity != "" {
				cs.refEntity = col.RefEntity

				if col.RefField == "" && len(col.NestedColumns) > 0 {
					// 完整嵌入引用（RefField=="" 且有 NestedColumns）：按 ID 查找單筆數據
					cs.isEmbedRef = true
					// 收集嵌套欄位名稱
					cs.nestedColumns = make([]string, len(col.NestedColumns))
					for i, nc := range col.NestedColumns {
						cs.nestedColumns[i] = nc.Name
					}
				} else if col.RefField != "" {
					// 字段引用（RefField!=""）：按字段值查找多筆數據
					// 字段引用不需要 NestedColumns，會嵌入整個被引用表的匹配記錄
					cs.isFieldRef = true
					cs.refField = col.RefField
				}
			}

			es.columns[col.Name] = cs
		}

		index[entity.Name] = es
	}

	return index
}

// WithImportMode 設置導入模式
func WithImportMode(mode ImportMode) Option {
	return func(s *Importer) {
		s.importMode = mode
	}
}

// WithKeywordSuffix 設定關鍵字衝突時的後綴（預設為 "_field"）
// 需與 SQL 生成器的設定保持一致
func WithKeywordSuffix(suffix string) Option {
	return func(s *Importer) {
		s.keywordSuffix = suffix
	}
}

// WithLayout 設定表頭版面（需與產生 schema 時使用的版面一致）
func WithLayout(layout source.Layout) Option {
	return func(s *Importer) {
		s.layout = layout
	}
}
