package wrapper

import (
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/schema"
)

func TestFormatMessageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Demo_Category", "DemoCategory"},
		{"Demo_ShopGoods", "DemoShopGoods"},
		{"User", "User"},
		{"user_info", "UserInfo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatMessageName(tt.input)
			if got != tt.want {
				t.Errorf("formatMessageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatWrapperName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Demo_Category", "DemoCategoryList"},
		{"Demo_ShopGoods", "DemoShopGoodsList"},
		{"User", "UserList"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatWrapperName(tt.input)
			if got != tt.want {
				t.Errorf("formatWrapperName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWrapperGenerator_Generate(t *testing.T) {
	tests := []struct {
		name              string
		entities          []schema.Entity
		packageName       string
		protoImportPath   string
		protoPackageAlias string
		functionName      string
		want              []string
		wantErr           bool
	}{
		{
			name: "基本生成",
			entities: []schema.Entity{
				{Name: "Demo_Category", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
			},
			protoImportPath:   "example.com/myapp/proto",
			protoPackageAlias: "pb",
			want: []string{
				"package envelope",
				`pb "example.com/myapp/proto"`,
				"func wrapEnvelope",
				`case "Demo_Category":`,
				"*pb.DemoCategoryList",
				"TableEnvelope_DemoCategoryList",
			},
			wantErr: false,
		},
		{
			name: "多個 entities",
			entities: []schema.Entity{
				{Name: "Demo_Category", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
				{Name: "Demo_Quest", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
				{Name: "Demo_ShopGoods", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
			},
			protoImportPath:   "example.com/myapp/proto",
			protoPackageAlias: "pb",
			want: []string{
				`case "Demo_Category":`,
				`case "Demo_Quest":`,
				`case "Demo_ShopGoods":`,
				"*pb.DemoCategoryList",
				"*pb.DemoQuestList",
				"*pb.DemoShopGoodsList",
			},
			wantErr: false,
		},
		{
			name: "跳過 wrapper entities",
			entities: []schema.Entity{
				{Name: "Demo_Category", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
				{Name: "Demo_CategoryList", Columns: []schema.Column{{Name: "items", RawType: "Demo_Category"}}, Metadata: schema.EntityMetadata{IsWrapper: true}},
			},
			protoImportPath:   "example.com/myapp/proto",
			protoPackageAlias: "pb",
			want: []string{
				`case "Demo_Category":`,
			},
			wantErr: false,
		},
		{
			name: "自訂 package name",
			entities: []schema.Entity{
				{Name: "User", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
			},
			packageName:       "mypackage",
			protoImportPath:   "example.com/myapp/proto",
			protoPackageAlias: "pb",
			want: []string{
				"package mypackage",
			},
			wantErr: false,
		},
		{
			name: "自訂 function name",
			entities: []schema.Entity{
				{Name: "User", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
			},
			protoImportPath:   "example.com/myapp/proto",
			protoPackageAlias: "pb",
			functionName:      "buildEnvelope",
			want: []string{
				"func buildEnvelope",
			},
			wantErr: false,
		},
		{
			name:     "空 entities 報錯",
			entities: []schema.Entity{},
			wantErr:  true,
		},
		{
			name: "只有 wrapper entities 報錯",
			entities: []schema.Entity{
				{Name: "Demo_CategoryList", Columns: []schema.Column{{Name: "items", RawType: "Demo_Category"}}, Metadata: schema.EntityMetadata{IsWrapper: true}},
			},
			protoImportPath:   "example.com/myapp/proto",
			protoPackageAlias: "pb",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []Option{}
			if tt.packageName != "" {
				opts = append(opts, WithPackageName(tt.packageName))
			}
			if tt.protoImportPath != "" {
				opts = append(opts, WithProtoPackage(tt.protoImportPath, tt.protoPackageAlias))
			}
			if tt.functionName != "" {
				opts = append(opts, WithFunctionName(tt.functionName))
			}

			g := New(opts...)
			got, err := g.Generate(tt.entities)

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := string(got)
			t.Logf("Generated wrapper:\n%s", output)

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing expected string: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestWrapperGenerator_Generate_NoWrapperEntityInOutput(t *testing.T) {
	entities := []schema.Entity{
		{Name: "Demo_Category", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
		{Name: "Demo_CategoryList", Columns: []schema.Column{{Name: "items", RawType: "Demo_Category"}}, Metadata: schema.EntityMetadata{IsWrapper: true}},
	}

	g := New(WithProtoPackage("example.com/myapp/proto", "pb"))
	got, err := g.Generate(entities)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := string(got)

	// 確認 wrapper entity 不會出現在 case 中
	if strings.Contains(output, `case "Demo_CategoryList"`) {
		t.Error("Output should not contain wrapper entity in case statement")
	}
}

func TestWrapperGenerator_Generate_EnvelopeName(t *testing.T) {
	entities := []schema.Entity{
		{Name: "User", Columns: []schema.Column{{Name: "id", RawType: "int32"}}},
	}

	tests := []struct {
		name string
		opts []Option
		want []string
	}{
		{
			name: "預設 envelope 名稱與 proto 生成器一致",
			opts: []Option{WithProtoPackage("example.com/myapp/proto", "pb")},
			want: []string{
				"(*pb.TableEnvelope, error)",
				"item.Payload = &pb.TableEnvelope_UserList{UserList: v}",
			},
		},
		{
			name: "自訂 envelope 名稱",
			opts: []Option{WithProtoPackage("example.com/myapp/proto", "pb"), WithEnvelopeName("Payload")},
			want: []string{
				"(*pb.Payload, error)",
				"&pb.Payload{",
				"item.Payload = &pb.Payload_UserList{UserList: v}",
			},
		},
		{
			name: "不使用 import 別名時直接引用型別",
			opts: []Option{WithProtoPackage("example.com/myapp/proto", "")},
			want: []string{
				`"example.com/myapp/proto"`,
				"(*TableEnvelope, error)",
				"msg.(*UserList)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.opts...).Generate(entities)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			output := string(got)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}
