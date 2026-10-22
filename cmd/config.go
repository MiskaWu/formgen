package cmd

import (
	"fmt"
	"os"

	"github.com/miskawu/formgen/internal/config"

	"github.com/spf13/cobra"
)

var (
	configInitOutputPath string
	configInitForce      bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理命令",
	Long:  `管理 formgen 配置檔案的相關操作。`,
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置檔案",
	Long: `初始化 formgen 配置檔案。

此命令會在當前目錄建立 formgen.yaml 配置檔案，
包含標準的類型映射定義。你可以根據專案需求修改此檔案。

範例：
  formgen config init                    # 在當前目錄建立 formgen.yaml
  formgen config init -o config.yaml     # 指定輸出路徑
  formgen config init --force            # 強制覆蓋已存在的檔案`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfigInit(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	},
}

func runConfigInit() error {
	outputPath := configInitOutputPath
	if outputPath == "" {
		outputPath = "formgen.yaml"
	}

	// 檢查檔案是否已存在
	if _, err := os.Stat(outputPath); err == nil {
		if !configInitForce {
			return fmt.Errorf("%s already exists, use --force to overwrite", outputPath)
		}
		fmt.Printf("Warning: overwriting existing %s\n", outputPath)
	}

	// 寫入配置檔
	if err := os.WriteFile(outputPath, []byte(config.DefaultConfig()), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created config file: %s\n", outputPath)
	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().StringVarP(&configInitOutputPath, "output", "o", "", "output config file path (default: formgen.yaml)")
	configInitCmd.Flags().BoolVarP(&configInitForce, "force", "f", false, "force overwrite existing file")
}
