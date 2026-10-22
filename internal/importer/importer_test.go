package importer

import (
	"testing"

	"github.com/miskawu/formgen/internal/source"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestImporter_WithOptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	var progressCalled bool
	s := New(db, "test.xlsx",
		WithBatchSize(200),
		WithProgressCallback(func(table string, current, total int) {
			progressCalled = true
		}),
	)

	if s.batchSize != 200 {
		t.Errorf("expected batch size 200, got %d", s.batchSize)
	}

	if s.onProgress == nil {
		t.Error("progress callback not set")
	}

	// 測試回調
	s.onProgress("test", 1, 10)
	if !progressCalled {
		t.Error("progress callback not called")
	}
}

func TestImporter_Import(t *testing.T) {
	// 創建內存資料庫
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 創建測試表（欄位名使用 snake_case，保留字加後綴 _field）
	err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT,
			value_field INTEGER
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// 準備測試數據
	sheet := source.Sheet{
		Name: "test_table",
		Rows: [][]string{
			{"註解", "註解", "註解"},
			{"ID", "Name", "Value"},
			{"int", "string", "int"},
			{"主鍵", "名稱", "數值"},
			{"", "", ""},
			{"1", "Test1", "100"},
			{"2", "Test2", "200"},
		},
	}

	// 創建 importer 並執行導入
	imp := New(db, "")
	if err := imp.Import(sheet); err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// 驗證資料
	var count int64
	db.Table("test_table").Count(&count)
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}

	var records []map[string]any
	db.Table("test_table").Find(&records)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0]["name"] != "Test1" {
		t.Errorf("expected name=Test1, got %v", records[0]["name"])
	}
}

func TestImporter_ImportSheet(t *testing.T) {
	// 創建內存資料庫
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 創建測試表（動態方式）
	err = db.Exec(`
		CREATE TABLE test_models (
			ID INTEGER PRIMARY KEY,
			Name TEXT,
			Age INTEGER
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// 創建 importer（無需註冊 model）
	s := New(db, "")

	// 模擬 Excel sheet 數據
	sheet := source.Sheet{
		Name: "test_models",
		Rows: [][]string{
			{"註解", "註解", "註解"},           // Row 0: 註解
			{"ID", "Name", "Age"},        // Row 1: 欄位名稱
			{"int64", "string", "int32"}, // Row 2: 欄位類型
			{"主鍵", "姓名", "年齡"},           // Row 3: 註解
			{"", "", ""},                 // Row 4: skip 標記
			{"1", "Alice", "25"},         // Row 5: 資料
			{"2", "Bob", "30"},           // Row 6: 資料
			{"3", "Charlie", "35"},       // Row 7: 資料
		},
	}

	// 執行導入
	err = db.Transaction(func(tx *gorm.DB) error {
		return s.importSheet(tx, sheet, "test_models")
	})

	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// 驗證資料（使用動態查詢）
	var count int64
	db.Table("test_models").Count(&count)
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}

	// 驗證具體資料（使用 map 接收）
	var records []map[string]any
	db.Table("test_models").Find(&records)

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	// 驗證第一筆資料
	firstRecord := records[0]
	if firstRecord["Name"] != "Alice" {
		t.Errorf("expected Name=Alice, got %v", firstRecord["Name"])
	}
}

func TestImporter_ImportSheet_WithSkip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 創建測試表
	err = db.Exec(`
		CREATE TABLE test_models (
			ID INTEGER PRIMARY KEY,
			Name TEXT,
			Age INTEGER
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	s := New(db, "")

	// 包含 skip 標記的 sheet
	sheet := source.Sheet{
		Name: "test_models",
		Rows: [][]string{
			{"註解", "註解", "註解", "註解"},
			{"ID", "Name", "Age", "Ignored"},
			{"int64", "string", "int32", "string"},
			{"主鍵", "姓名", "年齡", "忽略"},
			{"", "", "", "skip"}, // Age 之後的欄位被 skip
			{"1", "Alice", "25", "should_ignore"},
		},
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		return s.importSheet(tx, sheet, "test_models")
	})

	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// 使用 map slice 接收資料
	var records []map[string]any
	db.Table("test_models").Find(&records)

	if len(records) == 0 {
		t.Fatal("no records found")
	}

	record := records[0]
	if record["Name"] != "Alice" {
		t.Errorf("expected Name=Alice, got %v", record["Name"])
	}
}

func TestImporter_EmbeddedEntities_StillWritesToDB(t *testing.T) {
	// 這個測試驗證：即使是被嵌入的 Entity，數據仍會寫入 DB（DB 表是扁平的）
	// embeddedEntities 只影響 snapshot 生成，不影響 DB 導入

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 創建兩個測試表：主表和被嵌入的表
	err = db.Exec(`
		CREATE TABLE main_table (
			ID INTEGER PRIMARY KEY,
			Name TEXT
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create main_table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE embedded_table (
			id INTEGER PRIMARY KEY,
			value_field TEXT
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create embedded_table: %v", err)
	}

	// 不啟用 snapshot，embeddedEntities 不會生效
	s := New(db, "")

	// 主表的 sheet
	mainSheet := source.Sheet{
		Name: "main_table",
		Rows: [][]string{
			{"註解", "註解"},
			{"ID", "Name"},
			{"int64", "string"},
			{"主鍵", "名稱"},
			{"", ""},
			{"1", "Main1"},
		},
	}

	// 被嵌入表的 sheet
	embeddedSheet := source.Sheet{
		Name: "embedded_table",
		Rows: [][]string{
			{"註解", "註解"},
			{"ID", "Value"},
			{"int64", "string"},
			{"主鍵", "值"},
			{"", ""},
			{"1", "Embedded1"},
		},
	}

	// 導入兩個表
	if err := s.Import(mainSheet); err != nil {
		t.Fatalf("failed to import main_table: %v", err)
	}
	if err := s.Import(embeddedSheet); err != nil {
		t.Fatalf("failed to import embedded_table: %v", err)
	}

	// 驗證：兩個表的數據都應該被寫入 DB
	var mainCount int64
	db.Table("main_table").Count(&mainCount)
	if mainCount != 1 {
		t.Errorf("main_table: expected 1 record, got %d", mainCount)
	}

	var embeddedCount int64
	db.Table("embedded_table").Count(&embeddedCount)
	if embeddedCount != 1 {
		t.Errorf("embedded_table: expected 1 record, got %d", embeddedCount)
	}
}

func TestSnapshotConfig_EmbeddedEntities(t *testing.T) {
	// 測試 SnapshotConfig 中 EmbeddedEntities 的設置
	embedded := map[string]bool{
		"test_table": true,
	}

	config := SnapshotConfig{
		ProtoFilePath:    "nonexistent.proto", // 會導致 snapshot 初始化失敗
		EmbeddedEntities: embedded,
	}

	if config.EmbeddedEntities == nil {
		t.Error("EmbeddedEntities should not be nil")
	}

	if !config.EmbeddedEntities["test_table"] {
		t.Error("test_table should be marked as embedded")
	}
}
