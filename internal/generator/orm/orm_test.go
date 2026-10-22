package orm

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
		{"int32 to int32", "int32", "int32", true},
		{"int64 to int64", "int64", "int64", true},
		{"string to string", "string", "string", true},
		{"float32 to float32", "float32", "float32", true},
		{"float64 to float64", "float64", "float64", true},
		{"bool to bool", "bool", "bool", true},
		{"text to string", "text", "string", true},
		{"json to string", "json", "string", true},
		{"timestamp to time.Time", "timestamp", "time.Time", true},
		{"bytes to []byte", "bytes", "[]byte", true},
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
	mapper.AddMapping("customInt", "int64")
	mapper.AddMapping("userId", "string")

	got, ok := mapper.Map("customInt")
	if !ok || got != "int64" {
		t.Errorf("AddMapping failed: got %v, ok %v", got, ok)
	}

	got, ok = mapper.Map("userId")
	if !ok || got != "string" {
		t.Errorf("AddMapping failed: got %v, ok %v", got, ok)
	}
}

func TestDefaultTypeMapper_AddMappings(t *testing.T) {
	mapper := mapper.NewTypeMapper(defaultMappings)

	// 批量添加映射
	customMappings := map[string]string{
		"customType1": "int",
		"customType2": "string",
		"customType3": "bool",
	}
	mapper.AddMappings(customMappings)

	for src, expected := range customMappings {
		got, ok := mapper.Map(src)
		if !ok || got != expected {
			t.Errorf("AddMappings failed for %s: got %v, want %v", src, got, expected)
		}
	}
}

func TestORMGenerator_Generate(t *testing.T) {
	tests := []struct {
		name              string
		entity            schema.Entity
		customMappings    map[string]string // 自定義映射
		packageName       string            // 自定義 package name
		protoImportPath   string            // proto import 路徑
		protoPackageAlias string            // proto package 別名
		want              []string          // 期望在輸出中出現的字串
		wantErr           bool
	}{
		{
			name: "基本 GORM model 生成",
			entity: schema.Entity{
				Name: "User",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
					{Name: "age", RawType: "int32", IsArray: false},
				},
			},
			want: []string{
				"package models",
				"type User struct",
				"Id",
				"int64",
				"Name",
				"string",
				"Age",
				"int32",
				`gorm:"column:id;primaryKey"`,
				`gorm:"column:name"`,
				`gorm:"column:age"`,
				`json:"id"`,
				`json:"name"`,
				`json:"age"`,
				"func (User) TableName() string",
				`return "User"`,
			},
			wantErr: false,
		},
		{
			name: "帶陣列的 model",
			entity: schema.Entity{
				Name: "Post",
				Columns: []schema.Column{
					{Name: "postId", RawType: "int64", IsArray: false},
					{Name: "tags", RawType: "string", IsArray: true},
					{Name: "scores", RawType: "float32", IsArray: true},
				},
			},
			want: []string{
				"package models",
				"type Post struct",
				"PostId",
				"int64",
				"Tags",
				"[]string",
				"Scores",
				"[]float32",
				`gorm:"column:postId;primaryKey"`,
				`gorm:"column:tags;serializer:json"`,
				`gorm:"column:scores;serializer:json"`,
				"func (Post) TableName() string",
				`return "Post"`,
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
				"package models",
				"type Item struct",
				"ItemId",
				"int32",
				"Count",
				"int32",
				`gorm:"column:itemId;primaryKey"`,
				`gorm:"column:count"`,
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
					{Name: "createdAt", RawType: "timestamp", IsArray: false},
				},
			},
			want: []string{
				"package models",
				"type ComplexTable struct",
				"Id",
				"uint64",
				"Active",
				"bool",
				"Items",
				"[]int32",
				"Price",
				"float64",
				"Description",
				"string",
				"CreatedAt",
				"time.Time",
				`gorm:"column:id;primaryKey"`,
				`gorm:"column:active"`,
				`gorm:"column:items;serializer:json"`,
				`gorm:"column:price"`,
				`gorm:"column:description"`,
				`gorm:"column:createdAt"`,
			},
			wantErr: false,
		},
		{
			name: "bytes 類型陣列處理",
			entity: schema.Entity{
				Name: "BinaryData",
				Columns: []schema.Column{
					{Name: "id", RawType: "int32", IsArray: false},
					{Name: "data", RawType: "bytes", IsArray: false},
					{Name: "chunks", RawType: "bytes", IsArray: true},
				},
			},
			want: []string{
				"package models",
				"type BinaryData struct",
				"Id",
				"int32",
				"Data",
				"[]byte",
				"Chunks",
				"[][]byte",
				`gorm:"column:id;primaryKey"`,
				`gorm:"column:data"`,
				`gorm:"column:chunks;serializer:json"`,
			},
			wantErr: false,
		},
		{
			name: "自定義 package name",
			entity: schema.Entity{
				Name: "Product",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
				},
			},
			packageName: "entities",
			want: []string{
				"package entities",
				"type Product struct",
				"Id",
				"int64",
				"Name",
				"string",
			},
			wantErr: false,
		},
		{
			name: "生成 ToProto 方法",
			entity: schema.Entity{
				Name: "User",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
				},
			},
			protoImportPath:   "github.com/yourproject/proto",
			protoPackageAlias: "pb",
			want: []string{
				"package models",
				`import (`,
				`pb "github.com/yourproject/proto"`,
				"type User struct",
				"ToProto",
				"pb.User",
				"m.Id",
				"m.Name",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用預設選項或自定義映射
			var g gen.Generator
			opts := []Option{}
			if len(tt.customMappings) > 0 {
				opts = append(opts, WithCustomMappings(tt.customMappings))
			}
			if tt.packageName != "" {
				opts = append(opts, WithPackageName(tt.packageName))
			}
			if tt.protoImportPath != "" {
				opts = append(opts, WithProtoPackage(tt.protoImportPath, tt.protoPackageAlias))
			}
			g = New(opts...)

			got, err := g.Generate([]schema.Entity{tt.entity})

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := string(got)
			t.Logf("Generated ORM model:\n%s", output)

			// 檢查期望的字串是否都出現
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing expected string: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestBuildGormTag(t *testing.T) {
	gen := newGenerator()

	tests := []struct {
		name         string
		fieldName    string
		isPrimaryKey bool
		isArray      bool
		want         string
	}{
		{
			name:         "primary key",
			fieldName:    "id",
			isPrimaryKey: true,
			isArray:      false,
			want:         "column:id;primaryKey",
		},
		{
			name:         "regular field",
			fieldName:    "name",
			isPrimaryKey: false,
			isArray:      false,
			want:         "column:name",
		},
		{
			name:         "array field",
			fieldName:    "tags",
			isPrimaryKey: false,
			isArray:      true,
			want:         "column:tags;serializer:json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.buildGormTag(tt.fieldName, tt.isPrimaryKey, tt.isArray)
			if got != tt.want {
				t.Errorf("buildGormTag() = %q, want %q", got, tt.want)
			}
		})
	}
}
