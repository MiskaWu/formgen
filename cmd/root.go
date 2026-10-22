package cmd

import (
	"fmt"
	"os"

	"github.com/miskawu/formgen/internal/config"

	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "formgen",
	Short: "架構定義生成工具",
	Long: `formgen 是一個 Schema-First 的多格式代碼生成工具。
支援從 Excel 生成 YAML Schema，並從 YAML Schema 生成 SQL DDL、GORM model、
.proto 定義與 envelope 包裝函數；也能把 Excel 資料匯入資料庫並存成 protobuf 快照。

使用此工具可以簡化資料模型的定義和維護流程，確保多個系統之間的資料結構一致性。`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// initConfig 初始化配置系統
func initConfig() {
	if err := config.Init(cfgFile); err != nil {
		fmt.Printf("Warning: failed to load config: %v\n", err)
	}
}

func init() {
	// 在執行任何命令前初始化配置
	cobra.OnInitialize(initConfig)

	// 全域配置檔案旗標
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置檔案路徑 (預設搜尋 ./formgen.yaml, $HOME/formgen.yaml)")

	rootCmd.AddGroup(&cobra.Group{
		ID:    "generate",
		Title: "Generation Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "database",
		Title: "Database Commands",
	})

	schemaCmd.GroupID = "generate"
	sqlCmd.GroupID = "generate"
	ormCmd.GroupID = "generate"
	protoCmd.GroupID = "generate"
	wrapperCmd.GroupID = "generate"
	dbCmd.GroupID = "database"
}
