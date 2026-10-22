package migration

import "testing"

func TestEnsureMultiStatements(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		expected string
	}{
		{
			name:     "無參數的 DSN",
			dsn:      "user:pass@tcp(localhost:3306)/dbname",
			expected: "user:pass@tcp(localhost:3306)/dbname?multiStatements=true",
		},
		{
			name:     "已有其他參數的 DSN",
			dsn:      "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4",
			expected: "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&multiStatements=true",
		},
		{
			name:     "已包含 multiStatements=true",
			dsn:      "user:pass@tcp(localhost:3306)/dbname?multiStatements=true",
			expected: "user:pass@tcp(localhost:3306)/dbname?multiStatements=true",
		},
		{
			name:     "已包含 multiStatements=false（不修改）",
			dsn:      "user:pass@tcp(localhost:3306)/dbname?multiStatements=false",
			expected: "user:pass@tcp(localhost:3306)/dbname?multiStatements=false",
		},
		{
			name:     "多個參數且已包含 multiStatements",
			dsn:      "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&multiStatements=true&parseTime=true",
			expected: "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&multiStatements=true&parseTime=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureMultiStatements(tt.dsn)
			if result != tt.expected {
				t.Errorf("ensureMultiStatements(%q) = %q, want %q", tt.dsn, result, tt.expected)
			}
		})
	}
}
