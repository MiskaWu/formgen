package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	ormgen "github.com/miskawu/formgen/internal/generator/orm"
	yamlloader "github.com/miskawu/formgen/internal/schema/yaml"

	"github.com/spf13/cobra"
)

var (
	ormInput       string
	ormOutput      string
	ormPackageName string
	ormProtoPath   string
	ormProtoAlias  string
)

// ormCmd represents the orm command
var ormCmd = &cobra.Command{
	Use:   "orm",
	Short: "從 YAML Schema 生成 ORM Go 程式碼",
	Long: `從 YAML Schema 生成 ORM Go 程式碼。此命令會讀取 YAML Schema 檔案，
並生成對應的 Go 結構體和 GORM ORM 映射程式碼。

YAML Schema 作為中間格式，保留了細緻的類型資訊，
確保生成的 ORM model 類型精確。

範例：
  formgen orm --input schema.yaml --output models/models.go
  formgen orm -i schema.yaml -o models/models.go --package models
  formgen orm -i schema.yaml -o models/models.go --proto-path github.com/yourproject/api/proto --proto-alias pb`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runOrm(); err != nil {
			fmt.Printf("❌ 轉換失敗: %v\n", err)
			return
		}
		fmt.Printf("✅ Go 程式碼生成成功: %s\n", ormOutput)
	},
}

func runOrm() error {
	// 如果未指定輸出檔案，則使用預設名稱
	if ormOutput == "" {
		ormOutput = "models.go"
	}

	// 讀取 YAML Schema 檔案
	fmt.Printf("📖 讀取 YAML Schema: %s\n", ormInput)
	data, err := os.ReadFile(ormInput)
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

	// 生成 Go 代碼
	fmt.Printf("⚙️  生成 GORM model (%d 個 entity)...\n", len(entities))
	var genOpts []ormgen.Option
	genOpts = append(genOpts, ormgen.WithPackageName(ormPackageName))
	if ormProtoPath != "" {
		genOpts = append(genOpts, ormgen.WithProtoPackage(ormProtoPath, ormProtoAlias))
	}

	generator := ormgen.New(genOpts...)

	// 生成所有 entities 的代碼
	generatedBytes, err := generator.Generate(entities)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// 確保輸出目錄存在
	outputDir := filepath.Dir(ormOutput)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 寫入檔案
	fmt.Printf("💾 寫入檔案: %s\n", ormOutput)
	if err := os.WriteFile(ormOutput, generatedBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(ormCmd)

	// 定義命令旗標
	ormCmd.Flags().StringVarP(&ormInput, "input", "i", "schema.yaml", "輸入的 YAML Schema 檔案路徑")
	ormCmd.Flags().StringVarP(&ormOutput, "output", "o", "", "輸出的 Go 檔案路徑（預設：models.go）")
	ormCmd.Flags().StringVarP(&ormPackageName, "package", "p", "models", "生成的 Go 套件名稱")
	ormCmd.Flags().StringVar(&ormProtoPath, "proto-path", "", "Proto 套件路徑（例如：github.com/yourproject/api/proto）")
	ormCmd.Flags().StringVar(&ormProtoAlias, "proto-alias", "pb", "Proto 套件別名")
}
