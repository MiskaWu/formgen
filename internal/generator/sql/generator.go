package sql

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/miskawu/formgen/internal/mapper"
	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/strcase"
)

type generator struct {
	typeMapper    mapper.TypeMapper
	template      *template.Template
	keywordSuffix string
}

// newGenerator 創建 sql 生成器的內部實現
func newGenerator(opts ...Option) *generator {
	// 應用預設選項
	o := defaultOptions()

	// 應用使用者提供的選項
	for _, opt := range opts {
		opt(o)
	}

	// 解析模板
	tmpl, err := template.New("sql").Parse(sqlTemplate)
	if err != nil {
		panic("failed to parse sql template: " + err.Error())
	}

	return &generator{
		typeMapper:    o.typeMapper,
		template:      tmpl,
		keywordSuffix: o.keywordSuffix,
	}
}

// field 表示 SQL 表中的欄位
type field struct {
	Name         string
	Type         string
	IsPrimaryKey bool
	NotNull      bool
	Comment      string
}

// table 表示 SQL 表
type table struct {
	TableName string
	Fields    []field
}

// Generate 將 Entity 列表轉換為單一 SQL 檔案，包含多個 CREATE TABLE 語句
func (g *generator) Generate(entities []schema.Entity) ([]byte, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities to generate")
	}

	var result bytes.Buffer

	for i, entity := range entities {
		// 轉換欄位
		fields := make([]field, 0, len(entity.Columns))
		for j, col := range entity.Columns {
			var sqlType string
			var comment string

			// 結構陣列欄位（如 Loot[n].Field 格式）用 JSON 存儲
			if col.IsStruct {
				sqlType = "JSON"
				comment = "nested array field (stored as JSON)"
			} else {
				var err error
				sqlType, err = g.mapType(col.RawType, col.IsArray)
				if err != nil {
					return nil, fmt.Errorf("failed to map type for field %s in entity %s: %w", col.Name, entity.Name, err)
				}
				if col.IsArray {
					comment = "array field (stored as JSON)"
				}
			}

			// 第一個欄位預設為主鍵
			isPrimaryKey := (j == 0)

			fields = append(fields, field{
				Name:         g.formatColumnName(col.Name),
				Type:         sqlType,
				IsPrimaryKey: isPrimaryKey,
				NotNull:      isPrimaryKey, // 主鍵不能為 NULL
				Comment:      comment,
			})
		}

		// 構建表（表名轉為 snake_case）
		tbl := table{
			TableName: strcase.ToSnakeCase(entity.Name),
			Fields:    fields,
		}

		// 渲染模板
		var buf bytes.Buffer
		if err := g.template.Execute(&buf, tbl); err != nil {
			return nil, fmt.Errorf("failed to execute template for %s: %w", entity.Name, err)
		}

		// 添加到結果
		result.Write(buf.Bytes())

		// 在 CREATE TABLE 語句之間添加空行（除了最後一個）
		if i < len(entities)-1 {
			result.WriteString("\n")
		}
	}

	return result.Bytes(), nil
}

// formatColumnName 將欄位名轉為 snake_case 並檢查保留關鍵字
func (g *generator) formatColumnName(name string) string {
	snakeName := strcase.ToSnakeCase(name)
	if strcase.IsSQLReservedKeyword(snakeName) {
		return snakeName + g.keywordSuffix
	}
	return snakeName
}

// mapType 將來源類型映射到 SQL 類型
func (g *generator) mapType(rawType string, isArray bool) (string, error) {
	sqlType, ok := g.typeMapper.Map(rawType)
	if !ok {
		return "", fmt.Errorf("unknown type: %s", rawType)
	}

	// 處理陣列 - SQL 中可以用 JSON 儲存
	if isArray {
		return "JSON", nil
	}

	return sqlType, nil
}
