package wrapper

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"

	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/strcase"
)

type generator struct {
	packageName       string
	protoImportPath   string
	protoPackageAlias string
	functionName      string
	envelopeName      string
	template          *template.Template
}

// newGenerator 創建 wrapper 生成器的內部實現
func newGenerator(opts ...Option) *generator {
	// 應用預設選項
	o := defaultOptions()

	// 應用使用者提供的選項
	for _, opt := range opts {
		opt(o)
	}

	// 解析模板
	tmpl, err := template.New("wrapper").Parse(wrapperTemplate)
	if err != nil {
		panic("failed to parse wrapper template: " + err.Error())
	}

	return &generator{
		packageName:       o.packageName,
		protoImportPath:   o.protoImportPath,
		protoPackageAlias: o.protoPackageAlias,
		functionName:      o.functionName,
		envelopeName:      o.envelopeName,
		template:          tmpl,
	}
}

// caseEntry 表示 switch 中的單一 case
type caseEntry struct {
	EntityName  string // 原始 entity name: "Demo_ShopGoods"
	MessageName string // Proto message name: "DemoShopGoodsList"
}

// fileData 表示模板資料
type fileData struct {
	PackageName       string
	ProtoImportPath   string
	ProtoPackageAlias string
	FunctionName      string
	EnvelopeName      string
	Cases             []caseEntry
}

// Generate 將 Entity 列表轉換為 wrapper 函數程式碼
func (g *generator) Generate(entities []schema.Entity) ([]byte, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities to generate")
	}

	// 構建 case 列表，只處理非 wrapper entity
	cases := make([]caseEntry, 0, len(entities))
	for _, entity := range entities {
		if entity.Metadata.IsWrapper {
			continue
		}
		cases = append(cases, caseEntry{
			EntityName:  entity.Name,
			MessageName: formatWrapperName(entity.Name),
		})
	}

	if len(cases) == 0 {
		return nil, fmt.Errorf("no non-wrapper entities to generate")
	}

	// 構建檔案級模板資料
	data := fileData{
		PackageName:       g.packageName,
		ProtoImportPath:   g.protoImportPath,
		ProtoPackageAlias: g.protoPackageAlias,
		FunctionName:      g.functionName,
		EnvelopeName:      g.envelopeName,
		Cases:             cases,
	}

	// 執行模板
	var buf bytes.Buffer
	if err := g.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// 使用 go/format 格式化生成的程式碼
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format generated code: %w", err)
	}

	return formatted, nil
}

// formatMessageName 將名稱轉為 PascalCase（移除底線）
// 例如: Demo_RankReward -> DemoRankReward
func formatMessageName(name string) string {
	return strings.ReplaceAll(strcase.ToPascalCase(name), "_", "")
}

// formatWrapperName 生成 wrapper message 名稱
// 例如: Demo_RankReward -> DemoRankRewardList
func formatWrapperName(name string) string {
	return formatMessageName(name) + "List"
}
