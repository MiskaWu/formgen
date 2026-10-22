package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/miskawu/formgen/internal/migration"

	"github.com/spf13/cobra"
)

var (
	migrateDSN             string
	migrateDir             string
	migrateDatabase        string // 從 DSN 解析，非 flag
	migrateSteps           int
	migrateVersion         int
	migrateForce           bool
	migrateSQLFile         string
	migrateMultiStatements bool
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "執行資料庫遷移",
	Long: `執行資料庫架構遷移。此命令會根據定義的遷移腳本，
更新資料庫架構以符合最新的資料模型定義。

支援版本控制和回滾功能，確保資料庫架構變更的安全性。

子命令:
  up      - 執行所有待執行的遷移
  down    - 回滾所有遷移
  steps   - 執行指定步數的遷移
  version - 顯示當前遷移版本
  force   - 強制設置遷移版本
  create  - 創建新的遷移文件
  load    - 從 SQL 文件加載遷移`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "執行所有待執行的遷移",
	RunE:  runMigrateUp,
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "回滾所有遷移",
	RunE:  runMigrateDown,
}

var migrateStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "執行指定步數的遷移（正數向上，負數向下）",
	RunE:  runMigrateSteps,
}

var migrateVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "顯示當前遷移版本",
	RunE:  runMigrateVersion,
}

var migrateForceCmd = &cobra.Command{
	Use:   "force",
	Short: "強制設置遷移版本",
	RunE:  runMigrateForce,
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create [名稱]",
	Short: "創建新的遷移文件",
	Args:  cobra.ExactArgs(1),
	RunE:  runMigrateCreate,
}

var migrateLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "從 SQL 文件生成遷移文件",
	Long: `從現有的 SQL 文件（例如 output.sql）生成遷移文件。
會自動生成 UP 和 DOWN 遷移腳本。`,
	RunE: runMigrateLoad,
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	svc, err := createMigrationService()
	if err != nil {
		return err
	}
	defer svc.Close()

	fmt.Printf("執行遷移到最新版本...\n")
	if err := svc.Up(migrateDatabase); err != nil {
		return fmt.Errorf("遷移失敗: %w", err)
	}

	version, dirty, err := svc.Version(migrateDatabase)
	if err != nil {
		return err
	}

	fmt.Printf("✓ 遷移完成！當前版本: %d (dirty: %v)\n", version, dirty)
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	svc, err := createMigrationService()
	if err != nil {
		return err
	}
	defer svc.Close()

	fmt.Printf("回滾所有遷移...\n")
	if err := svc.Down(migrateDatabase); err != nil {
		return fmt.Errorf("回滾失敗: %w", err)
	}

	fmt.Printf("✓ 回滾完成！\n")
	return nil
}

func runMigrateSteps(cmd *cobra.Command, args []string) error {
	svc, err := createMigrationService()
	if err != nil {
		return err
	}
	defer svc.Close()

	fmt.Printf("執行 %d 步遷移...\n", migrateSteps)
	if err := svc.Steps(migrateDatabase, migrateSteps); err != nil {
		return fmt.Errorf("遷移失敗: %w", err)
	}

	version, dirty, err := svc.Version(migrateDatabase)
	if err != nil {
		return err
	}

	fmt.Printf("✓ 遷移完成！當前版本: %d (dirty: %v)\n", version, dirty)
	return nil
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	svc, err := createMigrationService()
	if err != nil {
		return err
	}
	defer svc.Close()

	version, dirty, err := svc.Version(migrateDatabase)
	if err != nil {
		return fmt.Errorf("獲取版本失敗: %w", err)
	}

	fmt.Printf("當前版本: %d\n", version)
	fmt.Printf("狀態: ")
	if dirty {
		fmt.Printf("dirty (需要修復)\n")
	} else {
		fmt.Printf("clean\n")
	}

	return nil
}

func runMigrateForce(cmd *cobra.Command, args []string) error {
	svc, err := createMigrationService()
	if err != nil {
		return err
	}
	defer svc.Close()

	fmt.Printf("強制設置版本為 %d...\n", migrateVersion)
	if err := svc.Force(migrateDatabase, migrateVersion); err != nil {
		return fmt.Errorf("強制設置版本失敗: %w", err)
	}

	fmt.Printf("✓ 版本已設置為 %d\n", migrateVersion)
	return nil
}

func runMigrateCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if migrateDir == "" {
		migrateDir = "./migrations"
	}

	fmt.Printf("創建遷移文件: %s\n", name)
	if err := migration.CreateMigrationFiles(migrateDir, name); err != nil {
		return fmt.Errorf("創建遷移文件失敗: %w", err)
	}

	fmt.Printf("✓ 遷移文件創建成功！\n")
	return nil
}

func runMigrateLoad(cmd *cobra.Command, args []string) error {
	if migrateDir == "" {
		migrateDir = "./migrations"
	}

	// 檢查 SQL 文件是否存在
	if _, err := os.Stat(migrateSQLFile); os.IsNotExist(err) {
		return fmt.Errorf("SQL 文件不存在: %s", migrateSQLFile)
	}

	fmt.Printf("從 %s 加載遷移...\n", migrateSQLFile)
	if err := migration.ConvertSQLToMigration(migrateSQLFile, migrateDir); err != nil {
		return fmt.Errorf("加載遷移失敗: %w", err)
	}

	fmt.Printf("✓ 遷移文件生成成功！位置: %s\n", migrateDir)
	return nil
}

func createMigrationService() (*migration.Service, error) {
	if migrateDSN == "" {
		return nil, fmt.Errorf("需要提供數據庫連接字符串 (--dsn)")
	}

	// 從 DSN 解析數據庫名稱
	migrateDatabase = parseDatabaseFromDSN(migrateDSN)
	if migrateDatabase == "" {
		return nil, fmt.Errorf("DSN 中需要包含數據庫名稱 (例如: user:pass@tcp(host:port)/dbname)")
	}

	if migrateDir == "" {
		migrateDir = "./migrations"
	}

	cfg := migration.Config{
		DSN:             migrateDSN,
		MigrationsDir:   migrateDir,
		DatabaseName:    migrateDatabase,
		MultiStatements: migrateMultiStatements,
	}

	return migration.NewService(cfg)
}

// parseDatabaseFromDSN 從 MySQL DSN 中解析數據庫名稱
// DSN 格式: user:password@tcp(host:port)/dbname?params
func parseDatabaseFromDSN(dsn string) string {
	// 找到 / 和 ? 之間的部分
	slashIdx := strings.LastIndex(dsn, "/")
	if slashIdx == -1 {
		return ""
	}

	dbPart := dsn[slashIdx+1:]

	// 移除查詢參數
	if qIdx := strings.Index(dbPart, "?"); qIdx != -1 {
		dbPart = dbPart[:qIdx]
	}

	return dbPart
}

func init() {
	dbCmd.AddCommand(migrateCmd)

	// 添加子命令
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStepsCmd)
	migrateCmd.AddCommand(migrateVersionCmd)
	migrateCmd.AddCommand(migrateForceCmd)
	migrateCmd.AddCommand(migrateCreateCmd)
	migrateCmd.AddCommand(migrateLoadCmd)

	// 全局標誌
	migrateCmd.PersistentFlags().StringVar(&migrateDSN, "dsn", "", "數據庫連接字符串 (例如: user:password@tcp(localhost:3306)/dbname)")
	migrateCmd.PersistentFlags().StringVar(&migrateDir, "dir", "./migrations", "遷移文件目錄")
	migrateCmd.PersistentFlags().BoolVar(&migrateMultiStatements, "multi-statements", true, "啟用多語句支援（執行包含多條 SQL 的遷移檔案）")

	// 注意：dsn 不設為全局必填，因為 load 和 create 子命令不需要數據庫連接。
	// 需要連接的命令會在 createMigrationService() 中檢查。

	// steps 命令標誌
	migrateStepsCmd.Flags().IntVar(&migrateSteps, "n", 1, "遷移步數（正數向上，負數向下）")

	// force 命令標誌
	migrateForceCmd.Flags().IntVar(&migrateVersion, "version", 0, "要強制設置的版本號")
	migrateForceCmd.MarkFlagRequired("version")

	// load 命令標誌
	migrateLoadCmd.Flags().StringVar(&migrateSQLFile, "file", "", "要加載的 SQL 文件路徑 (必填)")
	migrateLoadCmd.MarkFlagRequired("file")
}
