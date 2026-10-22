package config

import _ "embed"

//go:embed templates/default.yaml
var defaultConfigTemplate string

// DefaultConfig 返回預設配置模板內容
// 供 init 命令使用來建立配置檔案
func DefaultConfig() string {
	return defaultConfigTemplate
}
