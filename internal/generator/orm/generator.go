package orm

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"

	"github.com/miskawu/formgen/internal/mapper"
	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/strcase"
)

type generator struct {
	typeMapper        mapper.TypeMapper
	packageName       string
	protoImportPath   string
	protoPackageAlias string
	template          *template.Template
	keywordSuffix     string
}

// newGenerator 創建 ORM 生成器的內部實現
func newGenerator(opts ...Option) *generator {
	// 應用預設選項
	o := defaultOptions()

	// 應用使用者提供的選項
	for _, opt := range opts {
		opt(o)
	}

	// 解析模板
	tmpl, err := template.New("orm").Parse(ormTemplate)
	if err != nil {
		panic("failed to parse orm template: " + err.Error())
	}

	return &generator{
		typeMapper:        o.typeMapper,
		packageName:       o.packageName,
		protoImportPath:   o.protoImportPath,
		protoPackageAlias: o.protoPackageAlias,
		template:          tmpl,
		keywordSuffix:     o.keywordSuffix,
	}
}

// field 表示 GORM model 中的欄位
type field struct {
	Name    string
	Type    string
	GormTag string
	JsonTag string
}

// model 表示單個 GORM model
type model struct {
	StructName string
	TableName  string
	Fields     []field
}

// fileData 表示整個檔案的模板資料
type fileData struct {
	PackageName       string
	ProtoImportPath   string
	ProtoPackageAlias string
	HasProtoMethod    bool
	Models            []model
}

// Generate 將 Entity 列表轉換為單一 Go 檔案，包含多個 GORM models
func (g *generator) Generate(entities []schema.Entity) ([]byte, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities to generate")
	}

	// 構建模型列表
	models := make([]model, 0, len(entities))
	for _, entity := range entities {
		mdl, err := g.convertEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity %s: %w", entity.Name, err)
		}
		models = append(models, mdl)
	}

	// 構建檔案級模板資料
	data := fileData{
		PackageName:       g.packageName,
		ProtoImportPath:   g.protoImportPath,
		ProtoPackageAlias: g.protoPackageAlias,
		HasProtoMethod:    g.protoImportPath != "",
		Models:            models,
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

// convertEntity 將 schema.Entity 轉換為 model
func (g *generator) convertEntity(entity schema.Entity) (model, error) {
	// 轉換欄位
	fields := make([]field, 0, len(entity.Columns))
	for i, col := range entity.Columns {
		var goType string
		var isArrayField bool

		// 結構陣列欄位（如 Loot[n].Field 格式）用 []map[string]any 類型
		if col.IsStruct {
			goType = "[]map[string]any"
			isArrayField = true
		} else {
			var err error
			goType, err = g.mapType(col.RawType, col.IsArray)
			if err != nil {
				return model{}, fmt.Errorf("failed to map type for field %s: %w", col.Name, err)
			}
			isArrayField = col.IsArray
		}

		// 構建 GORM tag
		gormTag := g.buildGormTag(col.Name, i == 0, isArrayField)

		// JSON tag 使用 snake_case 格式
		jsonTag := strcase.ToSnakeCase(col.Name)

		// 轉換欄位名為 PascalCase（移除底線），並檢查是否需要轉義
		pascalName := strings.ReplaceAll(strcase.ToPascalCase(col.Name), "_", "")
		fieldName := g.escapeFieldName(pascalName)

		fields = append(fields, field{
			Name:    fieldName,
			Type:    goType,
			GormTag: gormTag,
			JsonTag: jsonTag,
		})
	}

	// 構建模型
	return model{
		StructName: strings.ReplaceAll(strcase.ToPascalCase(entity.Name), "_", ""),
		TableName:  entity.Name,
		Fields:     fields,
	}, nil
}

// escapeFieldName 檢查並轉義 Go 保留關鍵字
// 如果欄位名是保留關鍵字，則添加後綴
func (g *generator) escapeFieldName(name string) string {
	if IsReservedKeyword(name) {
		return name + g.keywordSuffix
	}
	return name
}

// mapType 將來源類型映射到 Go 類型
func (g *generator) mapType(rawType string, isArray bool) (string, error) {
	goType, ok := g.typeMapper.Map(rawType)
	if !ok {
		return "", fmt.Errorf("unknown type: %s", rawType)
	}

	// 處理陣列
	if isArray {
		return "[]" + goType, nil
	}

	return goType, nil
}

// buildGormTag 構建 GORM tag
func (g *generator) buildGormTag(name string, isPrimaryKey bool, isArray bool) string {
	tags := []string{}

	// 欄位名稱
	tags = append(tags, "column:"+name)

	// 主鍵標記
	if isPrimaryKey {
		tags = append(tags, "primaryKey")
	}

	// 陣列類型使用 serializer:json
	if isArray {
		tags = append(tags, "serializer:json")
	}

	return strings.Join(tags, ";")
}
