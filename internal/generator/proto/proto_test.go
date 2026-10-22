package proto

import (
	"context"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"

	"github.com/miskawu/formgen/internal/schema"
)

func TestIsReservedKeyword(t *testing.T) {
	tests := []struct {
		name string
		word string
		want bool
	}{
		{"message 是保留字", "message", true},
		{"MESSAGE 大寫也是保留字", "MESSAGE", true},
		{"service 是保留字", "service", true},
		{"repeated 是保留字", "repeated", true},
		{"optional 是保留字", "optional", true},
		{"username 不是保留字", "username", false},
		{"id 不是保留字", "id", false},
		{"name 不是保留字", "name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReservedKeyword(tt.word); got != tt.want {
				t.Errorf("IsReservedKeyword(%q) = %v, want %v", tt.word, got, tt.want)
			}
		})
	}
}

func TestProtoGenerator_Generate(t *testing.T) {
	tests := []struct {
		name    string
		entity  schema.Entity
		options []Option
		want    []string
		wantNot []string
	}{
		{
			name: "基本 message 生成",
			entity: schema.Entity{
				Name: "User",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
					{Name: "name", RawType: "string"},
					{Name: "age", RawType: "int32"},
				},
			},
			want: []string{
				"syntax = \"proto3\"",
				"message User",
				"int64 id = 1",
				"string name = 2",
				"int32 age = 3",
			},
		},
		{
			name: "帶陣列的 message",
			entity: schema.Entity{
				Name: "Post",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
					{Name: "tags", RawType: "string", IsArray: true},
				},
			},
			want: []string{
				"message Post",
				"int64 id = 1",
				"repeated string tags = 2",
			},
		},
		{
			name: "關鍵字轉義",
			entity: schema.Entity{
				Name: "Config",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
					{Name: "message", RawType: "string"},
					{Name: "service", RawType: "string"},
				},
			},
			want: []string{
				"string message_field = 2",
				"string service_field = 3",
			},
			wantNot: []string{
				"string message =", // 不應出現未轉義的 message
				"string service =", // 不應出現未轉義的 service
			},
		},
		{
			name: "自定義 package",
			entity: schema.Entity{
				Name: "Item",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
				},
			},
			options: []Option{
				WithPackageName("mypackage"),
				WithGoPackage("github.com/example/proto"),
			},
			want: []string{
				"package mypackage",
				"option go_package = \"github.com/example/proto\"",
			},
		},
		{
			name: "自定義關鍵字後綴",
			entity: schema.Entity{
				Name: "Test",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64"},
					{Name: "optional", RawType: "string"},
				},
			},
			options: []Option{
				WithKeywordSuffix("_value"),
			},
			want: []string{
				"string optional_value = 2",
			},
			wantNot: []string{
				"optional_field",
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
			t.Logf("Generated Proto:\n%s", output)

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

func TestProtoGenerator_Generate_MultipleEntities(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "User",
			Columns: []schema.Column{
				{Name: "id", RawType: "int64"},
				{Name: "name", RawType: "string"},
			},
		},
		{
			Name: "Product",
			Columns: []schema.Column{
				{Name: "productId", RawType: "int64"},
				{Name: "price", RawType: "float"},
			},
		},
	}

	gen := New(WithPackageName("test"))
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Proto:\n%s", output)

	want := []string{
		"message User",
		"message Product",
		"int64 id = 1",
		"string name = 2",
		"int64 product_id = 1", // Proto 使用 snake_case
		"float price = 2",
	}

	for _, w := range want {
		if !strings.Contains(output, w) {
			t.Errorf("Generate() missing: %q", w)
		}
	}
}

func TestProtoGenerator_Generate_WithWrapper(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "Demo_Category",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32"},
				{Name: "icon", RawType: "string"},
			},
		},
		{
			Name: "EventInfo",
			Columns: []schema.Column{
				{Name: "eventId", RawType: "int64"},
				{Name: "name", RawType: "string"},
			},
		},
	}

	gen := New(WithWrapper())
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Proto with Wrapper:\n%s", output)

	want := []string{
		// 原始 messages
		"message DemoCategory",
		"message EventInfo",
		// Wrapper messages（名稱 + List）
		"message DemoCategoryList",
		"repeated DemoCategory items = 1",
		"message EventInfoList",
		"repeated EventInfo items = 1",
	}

	for _, w := range want {
		if !strings.Contains(output, w) {
			t.Errorf("Generate() missing: %q", w)
		}
	}

	// 確保不會生成錯誤的 wrapper 名稱
	notWant := []string{
		"Demo_CategoryList", // 不應保留底線
		"Demo_Categorys",    // 不應直接加 s
	}

	for _, nw := range notWant {
		if strings.Contains(output, nw) {
			t.Errorf("Generate() should not contain: %q", nw)
		}
	}
}

func TestProtoGenerator_WithEnvelope(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "Demo_Category",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32"},
				{Name: "name", RawType: "string"},
			},
		},
		{
			Name: "Demo_Quest",
			Columns: []schema.Column{
				{Name: "questId", RawType: "int32"},
				{Name: "title", RawType: "string"},
			},
		},
	}

	gen := New(WithEnvelope(), WithGoPackage("./;proto"))
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Proto with envelope:\n%s", output)

	want := []string{
		// Entity messages
		"message DemoCategory",
		"message DemoQuest",
		// List wrappers
		"message DemoCategoryList",
		"message DemoQuestList",
		// Envelope structures
		"message TableEnvelope",
		"string key = 1",
		"string checksum = 2",
		"message TableBundle",
		"repeated TableEnvelope items = 2",
		// oneof payload
		"oneof payload",
		"DemoCategoryList demo_category_list = 10",
		"DemoQuestList demo_quest_list = 11",
	}

	for _, w := range want {
		if !strings.Contains(output, w) {
			t.Errorf("Generate() missing: %q", w)
		}
	}
}

func TestProtoGenerator_WithPayloadFieldStart(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "Item",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32"},
			},
		},
	}

	gen := New(WithEnvelope(), WithPayloadFieldStart(100))
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Proto:\n%s", output)

	if !strings.Contains(output, "ItemList item_list = 100") {
		t.Errorf("Expected field number 100, got:\n%s", output)
	}
}

func TestProtoGenerator_BackwardCompatibility(t *testing.T) {
	// 確保 WithWrapper() 不受影響
	entities := []schema.Entity{
		{
			Name: "User",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32"},
			},
		},
	}

	gen := New(WithWrapper()) // 只使用舊的 option
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Proto:\n%s", output)

	// 應該有 wrapper
	if !strings.Contains(output, "message UserList") {
		t.Errorf("Expected UserList wrapper")
	}

	// 不應該有 envelope 結構
	if strings.Contains(output, "TableEnvelope") || strings.Contains(output, "oneof") {
		t.Errorf("Should not contain envelope messages when only WithWrapper is used")
	}
}

func TestProtoGenerator_WithPluginTemplate_Default(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "Demo_Category",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32"},
				{Name: "name", RawType: "string"},
			},
		},
		{
			Name: "Demo_Quest",
			Columns: []schema.Column{
				{Name: "questId", RawType: "int32"},
				{Name: "title", RawType: "string"},
			},
		},
	}

	gen := New(WithPluginTemplate("default"), WithGoPackage("./;proto"))
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Plugin Proto:\n%s", output)

	want := []string{
		// Entity messages
		"message DemoCategory",
		"message DemoQuest",
		// List wrappers
		"message DemoCategoryList",
		"message DemoQuestList",
	}

	for _, w := range want {
		if !strings.Contains(output, w) {
			t.Errorf("Generate() missing: %q", w)
		}
	}

	// default 模板不應包含 envelope
	if strings.Contains(output, "TableEnvelope") {
		t.Errorf("default template should not contain TableEnvelope")
	}
}

func TestProtoGenerator_WithPluginTemplate_External(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "Demo_Category",
			Columns: []schema.Column{
				{Name: "id", RawType: "int32"},
				{Name: "name", RawType: "string"},
			},
		},
		{
			Name: "Demo_Quest",
			Columns: []schema.Column{
				{Name: "questId", RawType: "int32"},
				{Name: "title", RawType: "string"},
			},
		},
	}

	// 使用外部模板
	gen := New(WithPluginTemplate("../../../testdata/envelope_plugin.tmpl"), WithGoPackage("./;proto"))
	got, err := gen.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)
	t.Logf("Generated Plugin Proto with external template:\n%s", output)

	want := []string{
		// Entity messages
		"message DemoCategory",
		"message DemoQuest",
		// List wrappers
		"message DemoCategoryList",
		"message DemoQuestList",
		// 外部模板自訂的 envelope 結構
		"message Payload",
		"message PayloadBatch",
		// oneof payload
		"oneof payload",
		"DemoCategoryList demo_category_list = 10",
		"DemoQuestList demo_quest_list = 11",
	}

	for _, w := range want {
		if !strings.Contains(output, w) {
			t.Errorf("Generate() missing: %q", w)
		}
	}
}

// TestProtoGenerator_OutputCompiles 確認各種輸出模式產生的 .proto 都能被 protocompile 編譯
func TestProtoGenerator_OutputCompiles(t *testing.T) {
	entities := []schema.Entity{
		{
			Name: "Demo_ShopGoods",
			Columns: []schema.Column{
				{Name: "Id", RawType: "int32"},
				{Name: "GroupId", RawType: "int16"},
			},
		},
		{
			Name: "Demo_Shop",
			Columns: []schema.Column{
				{Name: "Id", RawType: "int32"},
				{Name: "GoodsGroupId", RawType: "int16", IsArray: true, IsReference: true, RefEntity: "Demo_ShopGoods", RefField: "GroupId"},
				{
					Name: "Loot", RawType: "LootStruct", IsArray: true, IsStruct: true,
					NestedColumns: []schema.Column{{Name: "ItemId", RawType: "int32"}, {Name: "Amount", RawType: "int32"}},
				},
			},
		},
	}

	tests := []struct {
		name string
		opts []Option
	}{
		{name: "基本", opts: nil},
		{name: "wrapper", opts: []Option{WithWrapper()}},
		{name: "envelope", opts: []Option{WithEnvelope()}},
		{name: "plugin default", opts: []Option{WithPluginTemplate("default")}},
		{name: "plugin 外部模板", opts: []Option{WithPluginTemplate("../../../testdata/envelope_plugin.tmpl")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := New(append(tt.opts, WithGoPackage("example.com/demo/pb;pb"))...).Generate(entities)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			compiler := protocompile.Compiler{
				Resolver: &protocompile.SourceResolver{
					Accessor: protocompile.SourceAccessorFromMap(map[string]string{"out.proto": string(out)}),
				},
			}
			if _, err := compiler.Compile(context.Background(), "out.proto"); err != nil {
				t.Errorf("generated proto does not compile: %v\n%s", err, out)
			}
		})
	}
}
