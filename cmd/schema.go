package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/miskawu/formgen/internal/config"
	yamlgen "github.com/miskawu/formgen/internal/generator/yaml"
	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/source"
	"github.com/miskawu/formgen/internal/source/excel"
	"github.com/miskawu/formgen/internal/source/resolver"

	"github.com/spf13/cobra"
)

var (
	schemaOutputPath string
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema [files...]",
	Short: "從 Excel 檔案生成 YAML Schema",
	Long: `從 Excel 檔案生成 YAML Schema。此命令會讀取一個或多個 Excel 檔案，
解析其內容，轉換為標準化的 YAML Schema 格式。

YAML Schema 保留了更細緻的類型資訊（如 int8, int16, uint32 等），
作為生成其他格式（Proto, SQL, ORM）的中間來源。

支援多檔案輸入，可使用 glob 模式（如 *.xlsx）。多檔案時會合併為單一 schema.yaml 檔案。

範例：
  formgen schema data/schema.xlsx
  formgen schema data/schema.xlsx --output api/schema.yaml
  formgen schema a.xlsx b.xlsx c.xlsx --output combined.yaml
  formgen schema data/*.xlsx --output api/schema.yaml`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		files, err := expandGlobPatterns(args)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}

		if err := runSchema(files); err != nil {
			fmt.Printf("❌ 轉換失敗: %v\n", err)
			return
		}
		fmt.Printf("✅ YAML Schema 生成成功: %s\n", schemaOutputPath)
	},
}

func runSchema(files []string) error {
	// 如果未指定輸出檔案，則根據輸入檔案決定
	if schemaOutputPath == "" {
		schemaOutputPath = defaultOutputPath(files, ".yaml", "schema.yaml")
	}

	// 讀取表頭版面設定
	layout, err := config.GetLayout()
	if err != nil {
		return err
	}

	// 解析所有 Excel 檔案並收集 entities
	loader := excel.NewLoader()
	parser := source.NewParser(source.WithLayout(layout))
	var entities []schema.Entity

	for _, file := range files {
		fmt.Printf("📊 讀取 Excel 檔案: %s\n", file)
		bytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read excel file %s: %w", file, err)
		}

		fmt.Printf("🔍 解析 Excel 數據: %s\n", file)
		sheets, err := loader.Load(bytes)
		if err != nil {
			return fmt.Errorf("failed to load excel %s: %w", file, err)
		}

		fmt.Printf("🔄 轉換為架構定義: %s\n", file)
		entity, err := parser.Parse(sheets[0])
		if err != nil {
			return fmt.Errorf("failed to parse schema from %s: %w", file, err)
		}

		entities = append(entities, entity)
	}

	// 解析參考類型（*entry 等）
	fmt.Printf("🔗 解析參考類型...\n")
	warnHandler := func(msg string) {
		fmt.Printf("  ⚠️  %s\n", msg)
	}
	res := resolver.New(entities, resolver.WithWarnHandler(warnHandler))
	resolved, err := res.Resolve()
	if err != nil {
		return fmt.Errorf("failed to resolve references: %w", err)
	}
	entities = resolved

	// 生成 YAML Schema
	fmt.Printf("⚙️  生成 YAML Schema (%d 個 entity)...\n", len(entities))

	// 從配置檔取得類型映射並注入 generator
	generator := yamlgen.New(yamlgen.WithMappings(config.GetCustomMappings()))
	generatedBytes, err := generator.Generate(entities)
	if err != nil {
		return fmt.Errorf("failed to generate yaml schema: %w", err)
	}

	// 確保輸出目錄存在
	if err := os.MkdirAll(filepath.Dir(schemaOutputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 寫入檔案
	fmt.Printf("💾 寫入檔案: %s\n", schemaOutputPath)
	if err := os.WriteFile(schemaOutputPath, generatedBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(schemaCmd)

	// 定義命令旗標
	schemaCmd.Flags().StringVarP(&schemaOutputPath, "output", "o", "", "輸出 YAML Schema 檔案路徑（單檔案預設：與輸入同名；多檔案預設：schema.yaml）")
}
