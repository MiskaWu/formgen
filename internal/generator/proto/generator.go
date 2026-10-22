package proto

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/miskawu/formgen/internal/mapper"
	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/strcase"
)

type generator struct {
	typeMapper        mapper.TypeMapper
	template          *template.Template
	packageName       string
	goPackage         string
	keywordSuffix     string
	generateWrapper   bool
	generateEnvelope  bool
	payloadFieldStart int
	pluginTemplate    string
}

//go:embed templates/proto.tmpl
var protoTemplate string

// newGenerator 創建 proto 生成器的內部實現
func newGenerator(opts ...Option) *generator {
	// 應用預設選項
	o := defaultOptions()

	// 應用使用者提供的選項
	for _, opt := range opts {
		opt(o)
	}

	// 解析模板
	tmpl, err := template.New("proto").Parse(protoTemplate)
	if err != nil {
		panic("failed to parse proto template: " + err.Error())
	}

	return &generator{
		typeMapper:        o.typeMapper,
		template:          tmpl,
		packageName:       o.packageName,
		goPackage:         o.goPackage,
		keywordSuffix:     o.keywordSuffix,
		generateWrapper:   o.generateWrapper,
		generateEnvelope:  o.generateEnvelope,
		payloadFieldStart: o.payloadFieldStart,
		pluginTemplate:    o.pluginTemplate,
	}
}

// protoField 表示 proto message 中的欄位
type protoField struct {
	Name   string
	Type   string
	Number int
}

// oneofField 表示 oneof 中的欄位
type oneofField struct {
	Name   string // 欄位名稱（snake_case）
	Type   string // 類型（如 DemoCategoryList）
	Number int    // 欄位編號
}

// oneofBlock 表示 oneof 區塊
type oneofBlock struct {
	Name   string       // oneof 名稱
	Fields []oneofField // oneof 中的欄位列表
}

// protoMessage 表示 proto message
type protoMessage struct {
	Name           string
	Fields         []protoField
	NestedMessages []protoMessage // 嵌套的 message 定義
	Oneofs         []oneofBlock   // oneof 區塊列表
}

// protoFile 表示整個 proto 檔案
type protoFile struct {
	PackageName      string
	GoPackage        string
	Messages         []protoMessage
	EnvelopeMessages []protoMessage // envelope 相關 messages（TableEnvelope、TableBundle）
}

// Generate 將 Entity 列表轉換為單一 Proto 檔案
func (g *generator) Generate(entities []schema.Entity) ([]byte, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities to generate")
	}

	// 如果使用 plugin 模式，委託給 plugin 生成器
	if g.pluginTemplate != "" {
		return g.generateWithPlugin(entities)
	}

	// 預估容量：啟用 wrapper 時每個 entity 會產生兩個 message
	capacity := len(entities)
	if g.generateWrapper {
		capacity *= 2
	}
	messages := make([]protoMessage, 0, capacity)

	for _, entity := range entities {
		// 跳過 wrapper 類型
		if entity.Metadata.IsWrapper {
			continue
		}

		msg, err := g.convertEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity %s: %w", entity.Name, err)
		}
		messages = append(messages, msg)

		// 如果啟用 wrapper 生成，直接在該 message 後面生成 List wrapper
		if g.generateWrapper {
			wrapper := g.generateWrapperMessage(msg)
			messages = append(messages, wrapper)
		}
	}

	// 構建 proto 檔案
	file := protoFile{
		PackageName: g.packageName,
		GoPackage:   g.goPackage,
		Messages:    messages,
	}

	// 如果啟用 envelope 輸出，添加 TableEnvelope / TableBundle
	if g.generateEnvelope {
		// 收集 wrapper 名稱用於生成 payload oneof
		wrapperNames := g.collectWrapperNames(entities)
		file.EnvelopeMessages = g.buildEnvelopeMessages(wrapperNames)
	}

	// 渲染模板
	var buf bytes.Buffer
	if err := g.template.Execute(&buf, file); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// generateWrapperMessage 為單一 message 生成對應的 List wrapper
// 例如: DemoCategory -> DemoCategoryList { repeated DemoCategory items = 1; }
func (g *generator) generateWrapperMessage(msg protoMessage) protoMessage {
	return protoMessage{
		Name: msg.Name + "List",
		Fields: []protoField{
			{
				Name:   "items",
				Type:   "repeated " + msg.Name,
				Number: 1,
			},
		},
	}
}

// convertEntity 將 schema.Entity 轉換為 protoMessage
func (g *generator) convertEntity(entity schema.Entity) (protoMessage, error) {
	fields := make([]protoField, 0, len(entity.Columns))
	nestedMessages := make([]protoMessage, 0)

	for i, col := range entity.Columns {
		field, nested, err := g.convertColumn(col, i+1)
		if err != nil {
			return protoMessage{}, fmt.Errorf("failed to convert field %s: %w", col.Name, err)
		}
		fields = append(fields, field)
		if nested != nil {
			nestedMessages = append(nestedMessages, *nested)
		}
	}

	return protoMessage{
		Name:           formatMessageName(entity.Name),
		Fields:         fields,
		NestedMessages: nestedMessages,
	}, nil
}

// convertColumn 將 schema.Column 轉換為 protoField，並處理嵌套 message
// 返回：欄位定義、嵌套 message（如果有）、錯誤
func (g *generator) convertColumn(col schema.Column, fieldNumber int) (protoField, *protoMessage, error) {
	// 處理參考類型
	if col.IsReference && col.RefEntity != "" {
		// 直接引用參考的 Entity 作為 message 類型
		refMsgName := formatMessageName(col.RefEntity)

		// 決定欄位類型
		fieldType := refMsgName
		if col.IsArray {
			fieldType = "repeated " + refMsgName
		}

		field := protoField{
			Name:   g.formatFieldName(col.Name),
			Type:   fieldType,
			Number: fieldNumber,
		}

		// 不生成嵌套 message，直接引用已存在的 message
		return field, nil, nil
	}

	// 處理結構陣列欄位（如 Loot[n].Field 格式）
	if col.IsStruct {
		// 生成嵌套 message
		nestedMsg, err := g.buildNestedMessage(col)
		if err != nil {
			return protoField{}, nil, fmt.Errorf("failed to build nested message: %w", err)
		}

		// 欄位類型為 repeated NestedMessageName
		fieldType := "repeated " + col.RawType

		field := protoField{
			Name:   g.formatFieldName(col.Name),
			Type:   fieldType,
			Number: fieldNumber,
		}

		return field, &nestedMsg, nil
	}

	// 普通欄位
	protoType, err := g.mapType(col.RawType, col.IsArray)
	if err != nil {
		return protoField{}, nil, fmt.Errorf("failed to map type: %w", err)
	}

	field := protoField{
		Name:   g.formatFieldName(col.Name),
		Type:   protoType,
		Number: fieldNumber,
	}

	return field, nil, nil
}

// buildNestedMessage 為結構陣列欄位生成嵌套 message
func (g *generator) buildNestedMessage(col schema.Column) (protoMessage, error) {
	fields := make([]protoField, 0, len(col.NestedColumns))

	for i, nested := range col.NestedColumns {
		protoType, err := g.mapType(nested.RawType, nested.IsArray)
		if err != nil {
			return protoMessage{}, fmt.Errorf("failed to map type for nested field %s: %w", nested.Name, err)
		}

		fields = append(fields, protoField{
			Name:   g.formatFieldName(nested.Name),
			Type:   protoType,
			Number: i + 1,
		})
	}

	return protoMessage{
		Name:   col.RawType, // 使用自動生成的類型名稱，如 "LootStruct"
		Fields: fields,
	}, nil
}

// formatFieldName 將欄位名轉為 snake_case 並檢查保留關鍵字
// Proto3 慣例使用 snake_case 欄位名
func (g *generator) formatFieldName(name string) string {
	snakeName := strcase.ToSnakeCase(name)
	if IsReservedKeyword(snakeName) {
		return snakeName + g.keywordSuffix
	}
	return snakeName
}

// formatMessageName 將名稱轉為 PascalCase（移除底線）
// 例如: Demo_RankReward -> DemoRankReward
func formatMessageName(name string) string {
	return strings.ReplaceAll(strcase.ToPascalCase(name), "_", "")
}

// mapType 將來源類型映射到 Proto 類型
func (g *generator) mapType(rawType string, isArray bool) (string, error) {
	protoType, ok := g.typeMapper.Map(rawType)
	if !ok {
		return "", fmt.Errorf("unknown type: %s", rawType)
	}

	// 處理陣列 - Proto 中使用 repeated
	if isArray {
		return "repeated " + protoType, nil
	}

	return protoType, nil
}

// collectWrapperNames 收集所有 wrapper 名稱
func (g *generator) collectWrapperNames(entities []schema.Entity) []string {
	names := make([]string, 0, len(entities))
	for _, entity := range entities {
		if !entity.Metadata.IsWrapper {
			msgName := formatMessageName(entity.Name)
			names = append(names, msgName+"List")
		}
	}
	return names
}

// EnvelopeMessageName 是 envelope 輸出格式中包裝單張表的 message 名稱
const EnvelopeMessageName = "TableEnvelope"

// BundleMessageName 是 envelope 輸出格式中包裝多張表的 message 名稱
const BundleMessageName = "TableBundle"

// buildEnvelopeMessages 構建 envelope 輸出格式的 messages
func (g *generator) buildEnvelopeMessages(wrapperNames []string) []protoMessage {
	return []protoMessage{
		g.buildEnvelopeMessage(wrapperNames),
		{
			Name: BundleMessageName,
			Fields: []protoField{
				{Name: "version", Type: "string", Number: 1},
				{Name: "items", Type: "repeated " + EnvelopeMessageName, Number: 2},
			},
		},
	}
}

// buildEnvelopeMessage 構建 TableEnvelope message（包含 payload oneof）
func (g *generator) buildEnvelopeMessage(wrapperNames []string) protoMessage {
	// 固定欄位：key 是 Entity 原名，checksum 讓接收端判斷內容是否變動
	fields := []protoField{
		{Name: "key", Type: "string", Number: 1},
		{Name: "checksum", Type: "string", Number: 2},
	}

	// 構建 payload oneof
	oneofFields := make([]oneofField, 0, len(wrapperNames))
	for i, wrapper := range wrapperNames {
		oneofFields = append(oneofFields, oneofField{
			Name:   strcase.ToSnakeCase(wrapper),
			Type:   wrapper,
			Number: g.payloadFieldStart + i,
		})
	}

	return protoMessage{
		Name:   EnvelopeMessageName,
		Fields: fields,
		Oneofs: []oneofBlock{
			{
				Name:   "payload",
				Fields: oneofFields,
			},
		},
	}
}
