package migration

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Service 提供數據庫遷移服務
type Service struct {
	db            *sql.DB
	migrationsDir string
}

// Config 遷移服務配置
type Config struct {
	DSN             string // 數據庫連接字符串
	MigrationsDir   string // 遷移文件目錄
	DatabaseName    string // 數據庫名稱
	MultiStatements bool   // 是否啟用多語句支援（預設 true）
}

// NewService 創建新的遷移服務實例
func NewService(cfg Config) (*Service, error) {
	dsn := cfg.DSN

	// 如果啟用多語句支援，確保 DSN 包含 multiStatements=true
	if cfg.MultiStatements {
		dsn = ensureMultiStatements(dsn)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Service{
		db:            db,
		migrationsDir: cfg.MigrationsDir,
	}, nil
}

// ensureMultiStatements 確保 DSN 包含 multiStatements=true 參數
// 這是 MySQL 執行包含多條語句的 migration 檔案所必需的
func ensureMultiStatements(dsn string) string {
	// 檢查是否已經包含 multiStatements
	if strings.Contains(dsn, "multiStatements=") {
		return dsn
	}

	// 根據是否已有參數來決定使用 ? 或 &
	if strings.Contains(dsn, "?") {
		return dsn + "&multiStatements=true"
	}
	return dsn + "?multiStatements=true"
}

// Close 關閉數據庫連接
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// getMigrate 獲取 migrate 實例
func (s *Service) getMigrate(databaseName string) (*migrate.Migrate, error) {
	driver, err := mysql.WithInstance(s.db, &mysql.Config{
		DatabaseName: databaseName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", s.migrationsDir)
	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"mysql",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

// Up 執行所有待執行的遷移
func (s *Service) Up(databaseName string) error {
	m, err := s.getMigrate(databaseName)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Down 回滾所有遷移
func (s *Service) Down(databaseName string) error {
	m, err := s.getMigrate(databaseName)
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}

// Steps 執行指定步數的遷移
func (s *Service) Steps(databaseName string, n int) error {
	m, err := s.getMigrate(databaseName)
	if err != nil {
		return err
	}

	if err := m.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d steps: %w", n, err)
	}

	return nil
}

// Version 獲取當前遷移版本
func (s *Service) Version(databaseName string) (uint, bool, error) {
	m, err := s.getMigrate(databaseName)
	if err != nil {
		return 0, false, err
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}

	return version, dirty, nil
}

// Force 強制設置版本（用於修復損壞的遷移狀態）
func (s *Service) Force(databaseName string, version int) error {
	m, err := s.getMigrate(databaseName)
	if err != nil {
		return err
	}

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}

	return nil
}

// CreateMigrationFiles 創建遷移文件
func CreateMigrationFiles(dir, name string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create migrations directory: %w", err)
		}
	}

	timestamp := getTimestamp()
	upFile := fmt.Sprintf("%s/%s_%s.up.sql", dir, timestamp, name)
	downFile := fmt.Sprintf("%s/%s_%s.down.sql", dir, timestamp, name)

	if err := os.WriteFile(upFile, []byte("-- Write your UP migration here\n"), 0644); err != nil {
		return fmt.Errorf("failed to create up migration file: %w", err)
	}

	if err := os.WriteFile(downFile, []byte("-- Write your DOWN migration here\n"), 0644); err != nil {
		return fmt.Errorf("failed to create down migration file: %w", err)
	}

	return nil
}

// getTimestamp 獲取時間戳用於遷移文件命名
func getTimestamp() string {
	return time.Now().Format("20060102150405")
}
