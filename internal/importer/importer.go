package importer

import (
	"fmt"

	"github.com/miskawu/formgen/internal/models"
	"github.com/miskawu/formgen/internal/serializer"
	"github.com/miskawu/formgen/internal/source"

	"gorm.io/gorm"
)

// ImportMode 定義導入模式
type ImportMode string

const (
	// ImportModeInsert 僅插入新數據，遇到主鍵衝突時失敗（默認）
	ImportModeInsert ImportMode = "insert"
	// ImportModeReplace 替換已存在的數據（DELETE + INSERT）
	ImportModeReplace ImportMode = "replace"
	// ImportModeUpsert 更新或插入（INSERT ... ON DUPLICATE KEY UPDATE）
	ImportModeUpsert ImportMode = "upsert"
	// ImportModeTruncate 導入前清空表（危險操作，僅適用於開發環境）
	ImportModeTruncate ImportMode = "truncate"
)

// Sheet 是 source.Sheet 的別名
type Sheet = source.Sheet

// ProgressCallback 用於報告導入進度
type ProgressCallback func(table string, current, total int)

// Importer 負責從 Excel 文件讀取數據並寫入資料庫
type Importer struct {
	db               *gorm.DB
	excelPath        string
	batchSize        int
	onProgress       ProgressCallback
	protoSerializer  *serializer.ProtoSerializer
	enableSnapshot   bool
	namespace        string
	importMode       ImportMode
	embeddedEntities map[string]bool // 被嵌入的 Entity 名稱集合（這些不需要單獨寫入 DB）
	keywordSuffix    string          // 關鍵字衝突時的後綴（預設為 "_field"）
	layout           source.Layout   // 表頭版面（預設 source.DefaultLayout()）

	// 嵌入數據支援
	entityIndex map[string]*entitySchema // Entity 名稱 -> schema（用於識別嵌入欄位）
	dataIndex   map[string]*tableData    // Entity 名稱 -> 該表的數據索引
}

// New 創建新的 Importer 實例
func New(db *gorm.DB, excelPath string, opts ...Option) *Importer {
	s := &Importer{
		db:            db,
		excelPath:     excelPath,
		batchSize:     100,              // 默認批次大小
		importMode:    ImportModeInsert, // 默認為 insert 模式
		keywordSuffix: "_field",         // 與 SQL 生成器保持一致
		layout:        source.DefaultLayout(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// 自動遷移 Snapshot 和 Version 表
	if err := db.AutoMigrate(&models.Snapshot{}, &models.Version{}); err != nil {
		fmt.Printf("Warning: failed to auto migrate snapshot/version tables: %v\n", err)
	}

	return s
}
