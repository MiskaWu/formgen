package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	protogen "github.com/miskawu/formgen/internal/generator/proto"
	yamlloader "github.com/miskawu/formgen/internal/schema/yaml"

	"github.com/spf13/cobra"
)

var (
	protoInput       string
	protoOutput      string
	protoPackageName string
	protoGoPackage   string
	protoWrapper     bool
	protoEnvelope    bool
	protoPlugin      string
)

// protoCmd represents the proto command
var protoCmd = &cobra.Command{
	Use:   "proto",
	Short: "從 YAML Schema 生成 Protocol Buffers 定義",
	Long: `從 YAML Schema 生成 Protocol Buffers 定義。此命令會讀取 YAML Schema 檔案，
並生成對應的 .proto 檔案，包含 message 定義。

YAML Schema 作為中間格式，保留了細緻的類型資訊，
確保生成的 Proto 類型精確匹配需求。

範例：
  formgen proto --input schema.yaml --output schema.proto
  formgen proto -i schema.yaml -o schema.proto --package myapp
  formgen proto -i schema.yaml -o schema.proto --go-package github.com/myorg/myapp/proto

  # 生成包含 wrapper message 的 proto（用於快照序列化）
  formgen proto -i schema.yaml -o schema.proto --wrapper

  # 另外生成 TableEnvelope（oneof payload）與 TableBundle
  formgen proto -i schema.yaml -o schema.proto --envelope

  # 使用內建 plugin 模板
  formgen proto -i schema.yaml -o tables.proto --plugin default --go-package "./;proto"

  # 使用自訂 plugin 模板
  formgen proto -i schema.yaml -o tables.proto --plugin mytemplate.tmpl --go-package "./;proto"`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runProto(); err != nil {
			fmt.Printf("❌ 轉換失敗: %v\n", err)
			return
		}
		fmt.Printf("✅ Proto 生成成功: %s\n", protoOutput)
	},
}

func runProto() error {
	// 如果未指定輸出檔案，則使用預設名稱
	if protoOutput == "" {
		base := filepath.Base(protoInput)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		dir := filepath.Dir(protoInput)
		protoOutput = filepath.Join(dir, name+".proto")
	}

	// 讀取 YAML Schema 檔案
	fmt.Printf("📖 讀取 YAML Schema: %s\n", protoInput)
	data, err := os.ReadFile(protoInput)
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

	// 生成 Proto 定義
	fmt.Printf("⚙️  生成 Proto schema (%d 個 entity)...\n", len(entities))
	var genOpts []protogen.Option
	if protoPackageName != "" {
		genOpts = append(genOpts, protogen.WithPackageName(protoPackageName))
	}
	if protoGoPackage != "" {
		genOpts = append(genOpts, protogen.WithGoPackage(protoGoPackage))
	}
	if protoPlugin != "" {
		genOpts = append(genOpts, protogen.WithPluginTemplate(protoPlugin))
		fmt.Printf("🔌 啟用 plugin 模式: %s\n", protoPlugin)
	} else if protoEnvelope {
		genOpts = append(genOpts, protogen.WithEnvelope())
		fmt.Println("📦 啟用 envelope 輸出（含 wrapper message）")
	} else if protoWrapper {
		genOpts = append(genOpts, protogen.WithWrapper())
		fmt.Println("📦 啟用 wrapper message 生成")
	}

	generator := protogen.New(genOpts...)

	// 生成所有 entities 的 Proto
	generatedBytes, err := generator.Generate(entities)
	if err != nil {
		return fmt.Errorf("failed to generate Proto: %w", err)
	}

	// 確保輸出目錄存在
	outputDir := filepath.Dir(protoOutput)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 寫入檔案
	fmt.Printf("💾 寫入檔案: %s\n", protoOutput)
	if err := os.WriteFile(protoOutput, generatedBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(protoCmd)

	// 定義命令旗標
	protoCmd.Flags().StringVarP(&protoInput, "input", "i", "schema.yaml", "輸入的 YAML Schema 檔案路徑")
	protoCmd.Flags().StringVarP(&protoOutput, "output", "o", "", "輸出的 Proto 檔案路徑（預設：與輸入同名但副檔名改為 .proto）")
	protoCmd.Flags().StringVarP(&protoPackageName, "package", "p", "", "Proto package 名稱（預設：schema）")
	protoCmd.Flags().StringVar(&protoGoPackage, "go-package", "", "Proto go_package 選項（例如：github.com/myorg/myapp/proto）")
	protoCmd.Flags().BoolVar(&protoWrapper, "wrapper", false, "生成 wrapper message（用於快照序列化，如 UserList 包含 repeated User）")
	protoCmd.Flags().BoolVar(&protoEnvelope, "envelope", false, "另外生成 TableEnvelope（oneof payload 涵蓋所有 List wrapper）與 TableBundle，會自動啟用 --wrapper")
	protoCmd.Flags().StringVar(&protoPlugin, "plugin", "", "使用 plugin 模板生成（'default' 使用內建模板，或指定 .tmpl 檔案路徑）")
}
