package proto

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/strcase"
)

//go:embed templates/plugin_default.tmpl
var pluginDefaultTemplate string

// PluginData 是傳遞給 plugin 模板的資料結構
type PluginData struct {
	PackageName   string         // proto package 名稱
	GoPackage     string         // go_package 選項
	Messages      []protoMessage // entity messages（不含 wrapper）
	Wrappers      []protoMessage // List wrapper messages
	PayloadFields []oneofField   // oneof payload 欄位
}

// generateWithPlugin 使用 plugin 模板生成 proto 檔案
func (g *generator) generateWithPlugin(entities []schema.Entity) ([]byte, error) {
	// 收集 entity messages（不含 wrapper）
	messages := make([]protoMessage, 0, len(entities))
	for _, entity := range entities {
		if entity.Metadata.IsWrapper {
			continue
		}
		msg, err := g.convertEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity %s: %w", entity.Name, err)
		}
		messages = append(messages, msg)
	}

	// 生成 wrapper messages
	wrappers := make([]protoMessage, 0, len(messages))
	for _, msg := range messages {
		wrappers = append(wrappers, g.generateWrapperMessage(msg))
	}

	// 生成 payload fields
	payloadFields := make([]oneofField, 0, len(wrappers))
	for i, wrapper := range wrappers {
		payloadFields = append(payloadFields, oneofField{
			Name:   strcase.ToSnakeCase(wrapper.Name),
			Type:   wrapper.Name,
			Number: g.payloadFieldStart + i,
		})
	}

	// 構建 plugin 資料
	data := PluginData{
		PackageName:   g.packageName,
		GoPackage:     g.goPackage,
		Messages:      messages,
		Wrappers:      wrappers,
		PayloadFields: payloadFields,
	}

	// 載入並解析模板
	tmpl, err := g.loadPluginTemplate()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin template: %w", err)
	}

	// 渲染模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute plugin template: %w", err)
	}

	return buf.Bytes(), nil
}

// loadPluginTemplate 載入 plugin 模板
func (g *generator) loadPluginTemplate() (*template.Template, error) {
	var tmplContent string

	if g.pluginTemplate == "default" {
		tmplContent = pluginDefaultTemplate
	} else {
		// 從檔案載入
		data, err := os.ReadFile(g.pluginTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to read plugin template file: %w", err)
		}
		tmplContent = string(data)
	}

	tmpl, err := template.New("plugin").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plugin template: %w", err)
	}

	return tmpl, nil
}
