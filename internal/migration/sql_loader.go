package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConvertSQLToMigration 將 SQL 文件轉換為遷移文件
func ConvertSQLToMigration(sqlFilePath, migrationsDir string) error {
	// 讀取 SQL 文件
	content, err := os.ReadFile(sqlFilePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	// 確保遷移目錄存在
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// 解析 SQL 語句
	statements := parseSQLStatements(string(content))

	if len(statements) == 0 {
		return fmt.Errorf("no SQL statements found in file")
	}

	// 生成遷移文件
	timestamp := getTimestamp()
	baseName := strings.TrimSuffix(filepath.Base(sqlFilePath), filepath.Ext(sqlFilePath))

	upFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.up.sql", timestamp, baseName))
	downFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.down.sql", timestamp, baseName))

	// 寫入 UP 遷移（CREATE 語句）
	if err := os.WriteFile(upFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write up migration: %w", err)
	}

	// 生成 DOWN 遷移（DROP 語句）
	downContent := generateDownMigration(statements)
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to write down migration: %w", err)
	}

	fmt.Printf("Created migration files:\n  - %s\n  - %s\n", upFile, downFile)

	return nil
}

// parseSQLStatements 解析 SQL 語句
func parseSQLStatements(content string) []string {
	// 移除注釋
	re := regexp.MustCompile(`--.*$`)
	content = re.ReplaceAllString(content, "")

	// 按分號分割語句
	statements := strings.Split(content, ";")

	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}

// generateDownMigration 生成 DOWN 遷移（回滾語句）
func generateDownMigration(statements []string) string {
	var downStatements []string

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		stmtUpper := strings.ToUpper(stmt)

		if strings.HasPrefix(stmtUpper, "CREATE TABLE") {
			// 提取表名
			tableName := extractTableName(stmt)
			if tableName != "" {
				downStatements = append(downStatements, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
			}
		}
	}

	// 反轉順序（先創建的後刪除）
	for i, j := 0, len(downStatements)-1; i < j; i, j = i+1, j-1 {
		downStatements[i], downStatements[j] = downStatements[j], downStatements[i]
	}

	if len(downStatements) == 0 {
		return "-- No DROP statements generated\n"
	}

	return strings.Join(downStatements, "\n") + "\n"
}

// extractTableName 從 CREATE TABLE 語句中提取表名
func extractTableName(stmt string) string {
	re := regexp.MustCompile(`CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)`)
	matches := re.FindStringSubmatch(stmt)

	if len(matches) > 1 {
		return strings.Trim(matches[1], "`\"'")
	}

	return ""
}
