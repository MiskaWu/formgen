package sql

import (
	"strings"
	"testing"

	gen "github.com/miskawu/formgen/internal/generator"
	"github.com/miskawu/formgen/internal/mapper"
	"github.com/miskawu/formgen/internal/schema"
)

func TestDefaultTypeMapper(t *testing.T) {
	mapper := mapper.NewTypeMapper(defaultMappings)

	tests := []struct {
		name       string
		sourceType string
		want       string
		wantOk     bool
	}{
		{"int32 to INT", "int32", "INT", true},
		{"int64 to BIGINT", "int64", "BIGINT", true},
		{"string to VARCHAR(255)", "string", "VARCHAR(255)", true},
		{"float32 to FLOAT", "float32", "FLOAT", true},
		{"float64 to DOUBLE", "float64", "DOUBLE", true},
		{"bool to BOOLEAN", "bool", "BOOLEAN", true},
		{"text to TEXT", "text", "TEXT", true},
		{"json to JSON", "json", "JSON", true},
		{"timestamp to TIMESTAMP", "timestamp", "TIMESTAMP", true},
		{"unknown type", "unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := mapper.Map(tt.sourceType)
			if ok != tt.wantOk {
				t.Errorf("Map() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if got != tt.want {
				t.Errorf("Map() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultTypeMapper_AddMapping(t *testing.T) {
	mapper := mapper.NewTypeMapper(defaultMappings)

	// 添加自定義映射
	mapper.AddMapping("customInt", "BIGINT")
	mapper.AddMapping("userId", "VARCHAR(50)")

	got, ok := mapper.Map("customInt")
	if !ok || got != "BIGINT" {
		t.Errorf("AddMapping failed: got %v, ok %v", got, ok)
	}

	got, ok = mapper.Map("userId")
	if !ok || got != "VARCHAR(50)" {
		t.Errorf("AddMapping failed: got %v, ok %v", got, ok)
	}
}

func TestDefaultTypeMapper_AddMappings(t *testing.T) {
	mapper := mapper.NewTypeMapper(defaultMappings)

	// 批量添加映射
	customMappings := map[string]string{
		"customType1": "INT",
		"customType2": "VARCHAR(100)",
		"customType3": "BOOLEAN",
	}
	mapper.AddMappings(customMappings)

	for src, expected := range customMappings {
		got, ok := mapper.Map(src)
		if !ok || got != expected {
			t.Errorf("AddMappings failed for %s: got %v, want %v", src, got, expected)
		}
	}
}

func TestSQLGenerator_Generate(t *testing.T) {
	tests := []struct {
		name           string
		entity         schema.Entity
		customMappings map[string]string // 自定義映射
		want           []string          // 期望在輸出中出現的字串
		wantErr        bool
	}{
		{
			name: "基本表生成",
			entity: schema.Entity{
				Name: "User",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
					{Name: "age", RawType: "int32", IsArray: false},
				},
			},
			want: []string{
				"CREATE TABLE user",
				"id BIGINT PRIMARY KEY NOT NULL",
				"name VARCHAR(255)",
				"age INT",
			},
			wantErr: false,
		},
		{
			name: "帶陣列的表",
			entity: schema.Entity{
				Name: "Post",
				Columns: []schema.Column{
					{Name: "postId", RawType: "int64", IsArray: false},
					{Name: "tags", RawType: "string", IsArray: true},
					{Name: "scores", RawType: "float32", IsArray: true},
				},
			},
			want: []string{
				"CREATE TABLE post",
				"post_id BIGINT PRIMARY KEY NOT NULL",
				"tags JSON",
				"scores JSON",
			},
			wantErr: false,
		},
		{
			name: "使用自定義映射",
			entity: schema.Entity{
				Name: "Item",
				Columns: []schema.Column{
					{Name: "itemId", RawType: "int32", IsArray: false},
					{Name: "count", RawType: "int32", IsArray: false},
				},
			},
			want: []string{
				"CREATE TABLE item",
				"item_id INT PRIMARY KEY NOT NULL",
				"count_field INT", // count 是 SQL 保留字，會加上後綴
			},
			wantErr: false,
		},
		{
			name: "混合類型",
			entity: schema.Entity{
				Name: "ComplexTable",
				Columns: []schema.Column{
					{Name: "id", RawType: "uint64", IsArray: false},
					{Name: "active", RawType: "bool", IsArray: false},
					{Name: "items", RawType: "int32", IsArray: true},
					{Name: "price", RawType: "float64", IsArray: false},
					{Name: "description", RawType: "text", IsArray: false},
				},
			},
			want: []string{
				"CREATE TABLE complex_table",
				"id BIGINT UNSIGNED PRIMARY KEY NOT NULL",
				"active BOOLEAN",
				"items JSON",
				"price DOUBLE",
				"description TEXT",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用預設選項或自定義映射
			var g gen.Generator
			if len(tt.customMappings) > 0 {
				g = New(WithCustomMappings(tt.customMappings))
			} else {
				g = New()
			}
			got, err := g.Generate([]schema.Entity{tt.entity})

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := string(got)
			t.Logf("Generated SQL:\n%s", output)

			// 檢查期望的字串是否都出現
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing expected string: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestSQLGenerator_WithCustomMappings(t *testing.T) {
	// 使用自定義映射
	customMappings := map[string]string{
		"userId":    "VARCHAR(50)",
		"timestamp": "BIGINT",
		"money":     "DECIMAL(10,2)",
	}

	// 使用 WithCustomMappings 選項
	gen := New(WithCustomMappings(customMappings))

	entity := schema.Entity{
		Name: "Transaction",
		Columns: []schema.Column{
			{Name: "userId", RawType: "userId", IsArray: false},
			{Name: "timestamp", RawType: "timestamp", IsArray: false},
			{Name: "amount", RawType: "money", IsArray: false},
		},
	}

	got, err := gen.Generate([]schema.Entity{entity})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated SQL with custom mappings:\n%s", output)

	expectations := []string{
		"CREATE TABLE transaction",
		"user_id VARCHAR(50) PRIMARY KEY NOT NULL",
		"timestamp_field BIGINT", // timestamp 是 SQL 資料類型關鍵字，會加上後綴
		"amount DECIMAL(10,2)",
	}

	for _, want := range expectations {
		if !strings.Contains(output, want) {
			t.Errorf("Output missing expected string: %q\nGot:\n%s", want, output)
		}
	}
}

func TestSQLGenerator_UnknownType(t *testing.T) {
	gen := New() // 使用預設選項

	entity := schema.Entity{
		Name: "TestEntity",
		Columns: []schema.Column{
			{Name: "field1", RawType: "unknownType", IsArray: false},
		},
	}

	_, err := gen.Generate([]schema.Entity{entity})
	if err == nil {
		t.Error("Generate() expected error for unknown type, got nil")
	}

	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("Generate() error message should mention 'unknown type', got: %v", err)
	}
}

// 完整示例測試
func TestSQLGenerator_FullExample(t *testing.T) {
	// 模擬從 YAML Schema 轉換來的 Entity（標準化類型）
	entity := schema.Entity{
		Name: "UserProfile",
		Columns: []schema.Column{
			{Name: "userId", RawType: "int32", IsArray: false},
			{Name: "userName", RawType: "string", IsArray: false},
			{Name: "emails", RawType: "string", IsArray: true},
			{Name: "age", RawType: "int32", IsArray: false},
			{Name: "balance", RawType: "float64", IsArray: false},
			{Name: "isActive", RawType: "bool", IsArray: false},
			{Name: "tags", RawType: "string", IsArray: true},
		},
	}

	gen := New()
	output, err := gen.Generate([]schema.Entity{entity})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result := string(output)
	t.Logf("\n=== Generated SQL File ===\n%s\n===========================", result)

	// 驗證關鍵元素
	checks := map[string]bool{
		"CREATE TABLE user_profile":        false,
		"user_id INT PRIMARY KEY NOT NULL": false,
		"user_name VARCHAR(255)":           false,
		"emails JSON":                      false, // array -> JSON
		"age INT":                          false,
		"balance DOUBLE":                   false,
		"is_active BOOLEAN":                false,
		"tags JSON":                        false, // array -> JSON
	}

	for check := range checks {
		if strings.Contains(result, check) {
			checks[check] = true
		}
	}

	// 報告檢查結果
	for check, found := range checks {
		if !found {
			t.Errorf("Missing expected content: %q", check)
		}
	}
}

// 測試 Options 模式
func TestSQLGenerator_WithOptions(t *testing.T) {
	t.Run("使用預設選項", func(t *testing.T) {
		gen := New() // 不傳入任何選項

		entity := schema.Entity{
			Name: "DefaultTest",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32", IsArray: false},
			},
		}

		output, err := gen.Generate([]schema.Entity{entity})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		result := string(output)
		if !strings.Contains(result, "id INT PRIMARY KEY NOT NULL") {
			t.Errorf("Expected default mapper to map int32 to INT and set as primary key")
		}
	})

	t.Run("使用 WithCustomMappings", func(t *testing.T) {
		gen := New(WithCustomMappings(map[string]string{
			"myType1": "BIGINT",
			"myType2": "BOOLEAN",
		}))

		entity := schema.Entity{
			Name: "CustomMappingsTest",
			Columns: []schema.Column{
				{Name: "field1", RawType: "myType1", IsArray: false},
				{Name: "field2", RawType: "myType2", IsArray: false},
			},
		}

		output, err := gen.Generate([]schema.Entity{entity})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		result := string(output)
		if !strings.Contains(result, "field1 BIGINT PRIMARY KEY NOT NULL") {
			t.Errorf("Expected myType1 to map to BIGINT and be primary key")
		}
		if !strings.Contains(result, "field2 BOOLEAN") {
			t.Errorf("Expected myType2 to map to BOOLEAN")
		}
	})

	t.Run("組合多個選項", func(t *testing.T) {
		// 雖然現在只有一個選項類型，但展示模式的可擴展性
		gen := New(
			WithCustomMappings(map[string]string{
				"special": "DECIMAL(10,2)",
			}),
			// 未來可以添加更多選項，如:
			// WithTableOptions(...),
			// WithIndexes(...),
		)

		entity := schema.Entity{
			Name: "MultiOptionsTest",
			Columns: []schema.Column{
				{Name: "price", RawType: "special", IsArray: false}, // 使用 price 而非 value（value 是保留字）
			},
		}

		output, err := gen.Generate([]schema.Entity{entity})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		result := string(output)
		if !strings.Contains(result, "price DECIMAL(10,2) PRIMARY KEY NOT NULL") {
			t.Errorf("Expected special to map to DECIMAL(10,2)")
		}
	})
}

// 測試陣列處理
func TestSQLGenerator_ArrayHandling(t *testing.T) {
	gen := New()

	entity := schema.Entity{
		Name: "ArrayTest",
		Columns: []schema.Column{
			{Name: "id", RawType: "int32", IsArray: false},
			{Name: "stringArray", RawType: "string", IsArray: true},
			{Name: "intArray", RawType: "int32", IsArray: true},
			{Name: "floatArray", RawType: "float32", IsArray: true},
		},
	}

	output, err := gen.Generate([]schema.Entity{entity})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result := string(output)
	t.Logf("Generated SQL with arrays:\n%s", result)

	// 所有陣列都應該映射為 JSON
	arrayChecks := []string{
		"string_array JSON",
		"int_array JSON",
		"float_array JSON",
	}

	for _, check := range arrayChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Expected array field to be mapped to JSON: %q", check)
		}
	}
}

// 測試主鍵設置
func TestSQLGenerator_PrimaryKey(t *testing.T) {
	gen := New()

	entity := schema.Entity{
		Name: "PrimaryKeyTest",
		Columns: []schema.Column{
			{Name: "userId", RawType: "int64", IsArray: false},
			{Name: "userName", RawType: "string", IsArray: false},
			{Name: "email", RawType: "string", IsArray: false},
		},
	}

	output, err := gen.Generate([]schema.Entity{entity})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result := string(output)
	t.Logf("Generated SQL with primary key:\n%s", result)

	// 第一個欄位應該是主鍵且 NOT NULL
	if !strings.Contains(result, "user_id BIGINT PRIMARY KEY NOT NULL") {
		t.Errorf("Expected first field to be primary key with NOT NULL constraint")
	}

	// 其他欄位不應該有 PRIMARY KEY
	if strings.Count(result, "PRIMARY KEY") != 1 {
		t.Errorf("Expected exactly one PRIMARY KEY, got: %s", result)
	}
}

// 測試特殊 SQL 類型
func TestSQLGenerator_SpecialTypes(t *testing.T) {
	gen := New()

	entity := schema.Entity{
		Name: "SpecialTypesTest",
		Columns: []schema.Column{
			{Name: "id", RawType: "int32", IsArray: false},
			{Name: "content", RawType: "text", IsArray: false},
			{Name: "metadata", RawType: "json", IsArray: false},
			{Name: "createdAt", RawType: "timestamp", IsArray: false},
		},
	}

	output, err := gen.Generate([]schema.Entity{entity})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result := string(output)
	t.Logf("Generated SQL with special types:\n%s", result)

	checks := []string{
		"content TEXT",
		"metadata JSON",
		"created_at TIMESTAMP",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Expected special type mapping: %q", check)
		}
	}
}
