package sql

import (
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/schema"
)

func TestSQLGenerator_KeywordEscaping(t *testing.T) {
	tests := []struct {
		name    string
		entity  schema.Entity
		options []Option
		want    []string
		wantNot []string
	}{
		{
			name: "基本保留字轉義",
			entity: schema.Entity{
				Name: "User",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
					{Name: "select", RawType: "string"},
					{Name: "from", RawType: "string"},
					{Name: "name", RawType: "string"},
				},
			},
			want: []string{
				"id BIGINT PRIMARY KEY NOT NULL",
				"select_field VARCHAR(255)",
				"from_field VARCHAR(255)",
				"name VARCHAR(255)",
			},
			wantNot: []string{
				"select VARCHAR", // 不應出現未轉義的 select
				"from VARCHAR",   // 不應出現未轉義的 from
			},
		},
		{
			name: "自定義後綴",
			entity: schema.Entity{
				Name: "Item",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
					{Name: "table", RawType: "string"},
				},
			},
			options: []Option{
				WithKeywordSuffix("_col"),
			},
			want: []string{
				"table_col VARCHAR(255)",
			},
			wantNot: []string{
				"table_field", // 不應使用預設後綴
			},
		},
		{
			name: "非保留字不轉義（欄位名轉為 snake_case）",
			entity: schema.Entity{
				Name: "Product",
				Columns: []schema.Column{
					{Name: "productId", RawType: "int64"},
					{Name: "productName", RawType: "string"},
					{Name: "price", RawType: "float32"},
				},
			},
			want: []string{
				"product_id BIGINT",
				"product_name VARCHAR(255)",
				"price FLOAT",
			},
			wantNot: []string{
				"_field", // 不應有任何轉義後綴
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(tt.options...)
			got, err := gen.Generate([]schema.Entity{tt.entity})
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			output := string(got)
			t.Logf("Generated SQL:\n%s", output)

			// 檢查應該出現的內容
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing expected: %q", want)
				}
			}

			// 檢查不應該出現的內容
			for _, notWant := range tt.wantNot {
				if strings.Contains(output, notWant) {
					t.Errorf("Generate() output should not contain: %q", notWant)
				}
			}
		})
	}
}
