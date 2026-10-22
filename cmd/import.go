package cmd

import (
	"fmt"
	"os"

	"github.com/miskawu/formgen/internal/config"
	"github.com/miskawu/formgen/internal/importer"
	"github.com/miskawu/formgen/internal/schema"
	yamlloader "github.com/miskawu/formgen/internal/schema/yaml"
	"github.com/miskawu/formgen/internal/source"
	"github.com/miskawu/formgen/internal/source/excel"
	"github.com/miskawu/formgen/internal/source/resolver"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	importDSN            string
	importBatchSize      int
	importMode           string
	importSnapshot       string // 快照命名空間（指定此參數啟用快照功能）
	importSnapshotProto  string // Proto 檔案路徑
	importSnapshotSchema string // YAML schema 路徑，用於分析嵌入關係
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import [files...]",
	Short: "將資料導入資料庫",
	Long: `將資料導入資料庫。此命令會讀取指定的 Excel 檔案，
並將資料批次導入到目標資料庫中。

支援事務處理，確保資料一致性。導入失敗時會自動回滾。
支援多檔案輸入，可使用 glob 模式（如 *.xlsx）。所有檔案在同一事務中處理。

嵌入關係處理：
  所有表的數據都會寫入資料庫（DB 表是扁平的）。
  被嵌入的 Entity（如 *entry 引用的表）不會單獨生成 snapshot，
  因為它們的數據已嵌入到父 Entity 的 protobuf 中。

  使用 --snapshot-schema 指定 YAML schema 檔案來分析嵌入關係。

範例：
  # 基本導入（單一檔案）
  formgen db import data/import.xlsx --dsn "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"

  # 使用 glob pattern 導入多個檔案
  formgen db import "data/*.xlsx" --dsn "..."

  # 導入多個檔案（所有 sheets 在同一事務中處理）
  formgen db import file1.xlsx file2.xlsx --dsn "..."

  # 重新導入（覆蓋已存在的數據）
  formgen db import data/import.xlsx --dsn "..." --mode upsert

  # 啟用快照功能
  formgen db import data/import.xlsx --dsn "..." \
    --snapshot default --snapshot-proto schema.proto --snapshot-schema schema.yaml

導入模式說明：
  insert   - 僅插入新數據，遇到主鍵衝突時失敗（默認）
  replace  - 替換已存在的數據（DELETE + INSERT）
  upsert   - 更新或插入（INSERT ... ON DUPLICATE KEY UPDATE）
  truncate - 導入前清空表（危險操作，僅適用於開發環境）`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		files, err := expandGlobPatterns(args)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}

		if err := runImport(files); err != nil {
			fmt.Printf("❌ 導入失敗: %v\n", err)
			return
		}
		fmt.Println("✅ 資料導入成功")
	},
}

func runImport(files []string) error {
	// 連接資料庫
	fmt.Printf("🔗 連接資料庫...\n")
	db, err := gorm.Open(mysql.Open(importDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// 獲取底層的 sql.DB 並設置 defer Close
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	defer sqlDB.Close()

	// 載入每個 Excel 檔案的第一個 sheet
	loader := excel.NewLoader()
	var sheets []source.Sheet

	for _, excelPath := range files {
		fmt.Printf("📊 讀取 Excel 檔案: %s\n", excelPath)
		data, err := os.ReadFile(excelPath)
		if err != nil {
			return fmt.Errorf("failed to read excel file %s: %w", excelPath, err)
		}

		loadedSheets, err := loader.Load(data)
		if err != nil {
			return fmt.Errorf("failed to load excel %s: %w", excelPath, err)
		}
		if len(loadedSheets) == 0 {
			return fmt.Errorf("no sheets found in excel %s", excelPath)
		}
		// 只取第一個 sheet（其他如 Description、更新紀錄 等為說明文件）
		sheets = append(sheets, loadedSheets[0])
	}

	fmt.Printf("🔍 共載入 %d 個檔案, %d 個 sheets\n", len(files), len(sheets))

	// 創建 importer 選項
	layout, err := config.GetLayout()
	if err != nil {
		return err
	}
	opts := []importer.Option{
		importer.WithLayout(layout),
		importer.WithBatchSize(importBatchSize),
		importer.WithProgressCallback(func(table string, current, total int) {
			fmt.Printf("  ⏳ 導入 %s: %d/%d (%.1f%%)\n",
				table, current, total, float64(current)/float64(total)*100)
		}),
	}

	// 設置導入模式
	mode := importer.ImportMode(importMode)
	switch mode {
	case importer.ImportModeInsert, importer.ImportModeReplace, importer.ImportModeUpsert, importer.ImportModeTruncate:
		opts = append(opts, importer.WithImportMode(mode))
		if mode == importer.ImportModeTruncate {
			fmt.Println("⚠️  警告: 使用 truncate 模式，將清空表中所有現有數據！")
		} else if mode != importer.ImportModeInsert {
			fmt.Printf("ℹ️  導入模式: %s\n", mode)
		}
	default:
		return fmt.Errorf("invalid import mode: %s (allowed: insert, replace, upsert, truncate)", importMode)
	}

	// 如果有指定 --snapshot，啟用快照功能
	var entities []schema.Entity
	var embeddedEntities map[string]bool

	if importSnapshot != "" {
		if importSnapshotProto == "" {
			return fmt.Errorf("--snapshot-proto is required when --snapshot is specified")
		}

		fmt.Printf("📸 啟用快照功能\n")
		fmt.Printf("  命名空間: %s\n", importSnapshot)
		fmt.Printf("  Proto 檔案: %s\n", importSnapshotProto)

		// 分析嵌入關係（只有啟用快照時才需要）
		if importSnapshotSchema != "" {
			fmt.Printf("  🔍 從 YAML schema 分析嵌入關係: %s\n", importSnapshotSchema)
			schemaData, err := os.ReadFile(importSnapshotSchema)
			if err != nil {
				return fmt.Errorf("failed to read schema file: %w", err)
			}
			entities, err = yamlloader.NewLoader().Load(schemaData)
			if err != nil {
				return fmt.Errorf("failed to load schema: %w", err)
			}
			// 找出被完整嵌入的 Entity（跳過 snapshot 生成）
			embeddedEntities = resolver.FindEmbeddedEntities(entities)
			if len(embeddedEntities) > 0 {
				fmt.Printf("  ℹ️  發現 %d 個被嵌入的 Entity（將跳過 snapshot 生成）:\n", len(embeddedEntities))
				for name := range embeddedEntities {
					fmt.Printf("      - %s\n", name)
				}
			}
		}

		opts = append(opts, importer.WithSnapshot(importer.SnapshotConfig{
			ProtoFilePath:    importSnapshotProto,
			Namespace:        importSnapshot,
			EmbeddedEntities: embeddedEntities,
			Entities:         entities,
		}))
	}

	// 創建 importer
	imp := importer.New(db, "", opts...)

	// 確保 snapshot/version 表存在（在事務外執行 DDL）
	if importSnapshot != "" {
		if err := imp.EnsureSnapshotTables(); err != nil {
			return fmt.Errorf("failed to ensure snapshot tables: %w", err)
		}
	}

	// 索引所有被引用的表（包含完整嵌入和字段引用）
	referencedEntities := resolver.FindReferencedEntities(entities)
	if len(referencedEntities) > 0 {
		fmt.Println("📇 索引被引用表的數據...")
		for _, sheet := range sheets {
			if referencedEntities[sheet.Name] {
				fmt.Printf("  索引 %s\n", sheet.Name)
				if err := imp.IndexSheetData(sheet); err != nil {
					return fmt.Errorf("failed to index sheet %s: %w", sheet.Name, err)
				}
			}
		}
	}

	// 執行導入（所有 sheets 在同一事務中處理，確保原子性）
	fmt.Println("🚀 開始導入資料...")
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, sheet := range sheets {
			if err := imp.ImportWithTx(tx, sheet); err != nil {
				return fmt.Errorf("failed to import %s: %w", sheet.Name, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func init() {
	dbCmd.AddCommand(importCmd)

	// 定義命令旗標
	importCmd.Flags().StringVar(&importDSN, "dsn", "", "資料庫連接字串 (必填)")
	importCmd.Flags().IntVar(&importBatchSize, "batch-size", 100, "批次處理大小")
	importCmd.Flags().StringVar(&importMode, "mode", "insert", "導入模式 (insert|replace|upsert|truncate)")

	// 快照相關旗標
	importCmd.Flags().StringVar(&importSnapshot, "snapshot", "", "快照命名空間（指定此參數啟用快照功能）")
	importCmd.Flags().StringVar(&importSnapshotProto, "snapshot-proto", "", "Proto 檔案路徑（啟用快照功能時必須指定）")
	importCmd.Flags().StringVar(&importSnapshotSchema, "snapshot-schema", "", "YAML schema 檔案路徑（用於分析嵌入關係）")

	// 標記必填參數
	importCmd.MarkFlagRequired("dsn")
}
