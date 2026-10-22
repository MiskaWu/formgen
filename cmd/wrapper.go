package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	wrappergen "github.com/miskawu/formgen/internal/generator/wrapper"
	yamlloader "github.com/miskawu/formgen/internal/schema/yaml"

	"github.com/spf13/cobra"
)

var (
	wrapperInput       string
	wrapperOutput      string
	wrapperPackageName string
	wrapperProtoPath   string
	wrapperProtoAlias  string
	wrapperFuncName    string
	wrapperEnvelope    string
)

// wrapperCmd represents the wrapper command
var wrapperCmd = &cobra.Command{
	Use:   "wrapper",
	Short: "從 YAML Schema 生成 envelope 包裝函數",
	Long: `從 YAML Schema 生成 envelope 包裝函數。此命令會讀取 YAML Schema 檔案，
並生成對應的 Go 原始碼檔案，包含 wrapEnvelope 函數。

此函數依資料表名稱把 proto.Message（各表的 XxxList）包進 envelope message 的
oneof payload，使用 switch-case 做靜態類型斷言。envelope message 由
「formgen proto --envelope」產生（預設名稱 TableEnvelope），或由自訂 plugin 模板定義。

範例：
  formgen wrapper --input schema.yaml --output envelope_gen.go --proto-path "example.com/myapp/proto" --proto-alias "pb"
  formgen wrapper -i schema.yaml -o wrapper.go --proto-path "example.com/myapp/proto" --package tables
  formgen wrapper -i schema.yaml -o wrapper.go --proto-path "example.com/myapp/proto" --proto-alias "pb" --func-name "buildEnvelope" --envelope "Payload"`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWrapper(); err != nil {
			fmt.Printf("❌ 轉換失敗: %v\n", err)
			return
		}
		fmt.Printf("✅ Wrapper 生成成功: %s\n", wrapperOutput)
	},
}

func runWrapper() error {
	// 驗證必填參數
	if wrapperProtoPath == "" {
		return fmt.Errorf("--proto-path is required")
	}

	// 如果未指定輸出檔案，則使用預設名稱
	if wrapperOutput == "" {
		dir := filepath.Dir(wrapperInput)
		wrapperOutput = filepath.Join(dir, "envelope_gen.go")
	}

	// 讀取 YAML Schema 檔案
	fmt.Printf("📖 讀取 YAML Schema: %s\n", wrapperInput)
	data, err := os.ReadFile(wrapperInput)
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

	// 生成 Wrapper 函數
	fmt.Printf("⚙️  生成 Wrapper 函數 (%d 個 entity)...\n", len(entities))
	var genOpts []wrappergen.Option
	if wrapperPackageName != "" {
		genOpts = append(genOpts, wrappergen.WithPackageName(wrapperPackageName))
	}
	if wrapperProtoPath != "" {
		genOpts = append(genOpts, wrappergen.WithProtoPackage(wrapperProtoPath, wrapperProtoAlias))
	}
	if wrapperFuncName != "" {
		genOpts = append(genOpts, wrappergen.WithFunctionName(wrapperFuncName))
	}
	if wrapperEnvelope != "" {
		genOpts = append(genOpts, wrappergen.WithEnvelopeName(wrapperEnvelope))
	}

	generator := wrappergen.New(genOpts...)

	// 生成 wrapper 函數
	generatedBytes, err := generator.Generate(entities)
	if err != nil {
		return fmt.Errorf("failed to generate wrapper: %w", err)
	}

	// 確保輸出目錄存在
	outputDir := filepath.Dir(wrapperOutput)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// 寫入檔案
	fmt.Printf("💾 寫入檔案: %s\n", wrapperOutput)
	if err := os.WriteFile(wrapperOutput, generatedBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(wrapperCmd)

	// 定義命令旗標
	wrapperCmd.Flags().StringVarP(&wrapperInput, "input", "i", "schema.yaml", "輸入的 YAML Schema 檔案路徑")
	wrapperCmd.Flags().StringVarP(&wrapperOutput, "output", "o", "", "輸出的 Go 檔案路徑（預設：與輸入同目錄的 envelope_gen.go）")
	wrapperCmd.Flags().StringVarP(&wrapperPackageName, "package", "p", "", "Go package 名稱（預設：envelope）")
	wrapperCmd.Flags().StringVar(&wrapperProtoPath, "proto-path", "", "Proto package 的 import 路徑（必填，例如：example.com/myapp/proto）")
	wrapperCmd.Flags().StringVar(&wrapperProtoAlias, "proto-alias", "", "Proto package 的別名（例如：pb）")
	wrapperCmd.Flags().StringVar(&wrapperFuncName, "func-name", "", "生成的函數名稱（預設：wrapEnvelope）")
	wrapperCmd.Flags().StringVar(&wrapperEnvelope, "envelope", "", "envelope message 名稱（預設：TableEnvelope）")
}
