package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	sqlgen "github.com/miskawu/formgen/internal/generator/sql"
	yamlloader "github.com/miskawu/formgen/internal/schema/yaml"

	"github.com/spf13/cobra"
)

var (
	sqlInput  string
	sqlOutput string
)

// sqlCmd represents the sql command
var sqlCmd = &cobra.Command{
	Use:   "sql",
	Short: "從 YAML Schema 生成 SQL 架構",
	Long: `從 YAML Schema 生成 SQL 資料庫架構。此命令會讀取 YAML Schema 檔案，
並生成對應的 SQL CREATE TABLE 語句。

YAML Schema 作為中間格式，保留了細緻的類型資訊，
確保生成的 SQL 類型精確匹配需求。

範例：
  formgen sql --input schema.yaml --output schema.sql
  formgen sql -i schema.yaml -o schema.sql`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSql(); err != nil {
			fmt.Printf("❌ 轉換失敗: %v\n", err)
			return
		}
		fmt.Printf("✅ SQL 生成成功: %s\n", sqlOutput)
	},
}

func runSql() error {
	// 如果未指定輸出檔案，則使用預設名稱
	if sqlOutput == "" {
		base := filepath.Base(sqlInput)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		dir := filepath.Dir(sqlInput)
		sqlOutput = filepath.Join(dir, name+".sql")
	}

	// 讀取 YAML Schema 檔案
	fmt.Printf("📖 讀取 YAML Schema: %s\n", sqlInput)
	data, err := os.ReadFile(sqlInput)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// 解析 YAML Schema
	fmt.Println("🔍 解析 YAML Schema...")
	loader := yamlloader.NewLoader()
	entities, err := loader.Load(data)
	if err != nil {
		return fmt.Errorf("failed to load yaml schema: %w", err)
	}

	// 生成 SQL 定義
	fmt.Printf("⚙️  生成 SQL schema (%d 個 entity)...\n", len(entities))
	generator := sqlgen.New()

	// 生成所有 entities 的 SQL
	generatedBytes, err := generator.Generate(entities)
	if err != nil {
		return fmt.Errorf("failed to generate SQL: %w", err)
	}

	// 確保輸出目錄存在
	outputDir := filepath.Dir(sqlOutput)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 寫入檔案
	fmt.Printf("💾 寫入檔案: %s\n", sqlOutput)
	if err := os.WriteFile(sqlOutput, generatedBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(sqlCmd)

	// 定義命令旗標
	sqlCmd.Flags().StringVarP(&sqlInput, "input", "i", "schema.yaml", "輸入的 YAML Schema 檔案路徑")
	sqlCmd.Flags().StringVarP(&sqlOutput, "output", "o", "", "輸出的 SQL 檔案路徑（預設：與輸入同名但副檔名改為 .sql）")
}
