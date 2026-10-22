package yaml

import (
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/schema"
)

// testMappings 提供測試用的標準類型映射
func testMappings() map[string]string {
	return map[string]string{
		// 整數類型
		"tinyint":  "int8",
		"int8":     "int8",
		"shortint": "int16",
		"int16":    "int16",
		"int":      "int32",
		"int32":    "int32",
		"int64":    "int64",

		// 無號整數類型
		"uint8":  "uint8",
		"uint16": "uint16",
		"uint":   "uint32",
		"uint32": "uint32",
		"uint64": "uint64",

		// 浮點數類型
		"float":   "float32",
		"float32": "float32",
		"float64": "float64",
		"double":  "float64",

		// 字符串和布爾
		"string": "string",
		"bool":   "bool",
		"byte":   "bytes",
		"bytes":  "bytes",

		// 時間類型
		"timestamp": "timestamp",
		"datetime":  "datetime",
		"date":      "date",
		"time":      "time",

		// 文字類型
		"text": "text",
		"json": "json",

		// 其他
		"any": "any",
		"map": "map",
	}
}

func TestYAMLGenerator_Generate(t *testing.T) {
	tests := []struct {
		name    string
		entity  schema.Entity
		options []Option
		want    []string
		wantErr bool
	}{
		{
			name: "基本欄位生成",
			entity: schema.Entity{
				Name: "User",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
				},
			},
			options: []Option{WithMappings(testMappings())},
			want: []string{
				"version: \"1\"",
				"name: User",
				"name: id",
				"type: int64",
				"name: name",
				"type: string",
			},
			wantErr: false,
		},
		{
			name: "細粒度整數類型",
			entity: schema.Entity{
				Name: "Score",
				Columns: []schema.Column{
					{Name: "tiny", RawType: "tinyint", IsArray: false},
					{Name: "short", RawType: "shortint", IsArray: false},
					{Name: "normal", RawType: "int", IsArray: false},
					{Name: "big", RawType: "int64", IsArray: false},
				},
			},
			options: []Option{WithMappings(testMappings())},
			want: []string{
				"name: Score",
				"type: int8",  // tinyint -> int8
				"type: int16", // shortint -> int16
				"type: int32", // int -> int32
				"type: int64", // int64 -> int64
			},
			wantErr: false,
		},
		{
			name: "陣列欄位",
			entity: schema.Entity{
				Name: "Post",
				Columns: []schema.Column{
					{Name: "tags", RawType: "string", IsArray: true},
				},
			},
			options: []Option{WithMappings(testMappings())},
			want: []string{
				"name: tags",
				"type: string",
				"isArray: true",
			},
			wantErr: false,
		},
		{
			name: "自定義類型映射",
			entity: schema.Entity{
				Name: "Item",
				Columns: []schema.Column{
					{Name: "entry_id", RawType: "entry", IsArray: false},
					{Name: "asset_path", RawType: "asset", IsArray: false},
				},
			},
			options: []Option{
				WithMappings(map[string]string{
					"entry": "int32",
					"asset": "string",
				}),
			},
			want: []string{
				"name: Item",
				"name: entry_id",
				"type: int32",
				"name: asset_path",
				"type: string",
			},
			wantErr: false,
		},
		{
			name: "未知類型應報錯",
			entity: schema.Entity{
				Name: "Invalid",
				Columns: []schema.Column{
					{Name: "field", RawType: "unknownType", IsArray: false},
				},
			},
			options: []Option{WithMappings(testMappings())},
			wantErr: true,
		},
		{
			name: "使用配置檔映射（含自定義業務類型）",
			entity: schema.Entity{
				Name: "DefaultMapping",
				Columns: []schema.Column{
					{Name: "field", RawType: "int64", IsArray: false},
					{Name: "flag", RawType: "tag", IsArray: false},
					{Name: "path", RawType: "asset", IsArray: false},
				},
			},
			// 模擬配置檔的完整映射（包含自定義業務類型）
			options: []Option{
				WithMappings(map[string]string{
					"int64": "int64",
					"tag":   "tag",
					"asset": "string",
				}),
			},
			want: []string{
				"name: DefaultMapping",
				"type: int64",
				"type: tag",
				"type: string",
			},
			wantErr: false,
		},
		{
			name: "浮點數類型",
			entity: schema.Entity{
				Name: "Metrics",
				Columns: []schema.Column{
					{Name: "rate", RawType: "float", IsArray: false},
					{Name: "precision", RawType: "double", IsArray: false},
				},
			},
			options: []Option{WithMappings(testMappings())},
			want: []string{
				"type: float32", // float -> float32
				"type: float64", // double -> float64
			},
			wantErr: false,
		},
		{
			name: "時間類型",
			entity: schema.Entity{
				Name: "Event",
				Columns: []schema.Column{
					{Name: "created_at", RawType: "timestamp", IsArray: false},
					{Name: "event_date", RawType: "date", IsArray: false},
				},
			},
			options: []Option{WithMappings(testMappings())},
			want: []string{
				"type: timestamp",
				"type: date",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(tt.options...)
			got, err := gen.Generate([]schema.Entity{tt.entity})

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := string(got)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing expected string: %q", want)
				}
			}

			t.Logf("Generated output:\n%s", output)
		})
	}
}

func TestYAMLGenerator_Generate_MultipleEntities(t *testing.T) {
	tests := []struct {
		name     string
		entities []schema.Entity
		options  []Option
		want     []string
		wantErr  bool
	}{
		{
			name: "多個 Entity",
			entities: []schema.Entity{
				{
					Name: "User",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64", IsArray: false},
					},
				},
				{
					Name: "Post",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64", IsArray: false},
						{Name: "user_id", RawType: "int64", IsArray: false},
					},
				},
			},
			options: []Option{WithMappings(testMappings())},
			want: []string{
				"version: \"1\"",
				"name: User",
				"name: Post",
				"name: user_id",
			},
			wantErr: false,
		},
		{
			name:     "空 entities",
			entities: []schema.Entity{},
			options:  []Option{WithMappings(testMappings())},
			want: []string{
				"version: \"1\"",
				"entities: []",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(tt.options...)
			got, err := gen.Generate(tt.entities)

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := string(got)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing expected string: %q", want)
				}
			}

			t.Logf("Generated output:\n%s", output)
		})
	}
}
