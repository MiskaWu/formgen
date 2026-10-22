// Package config 提供配置管理功能
// 支援環境變數優先，然後是配置檔案
package config

import (
	"fmt"
	"strings"

	"github.com/miskawu/formgen/internal/source"

	"github.com/spf13/viper"
)

// Config 代表整個應用程式的配置
type Config struct {
	Mappings MappingsConfig `mapstructure:"mappings"`
	Layout   LayoutConfig   `mapstructure:"layout"`
}

// MappingsConfig 代表類型映射的配置
// 使用者可以根據不同來源定義自己的類型映射
type MappingsConfig struct {
	// Custom 是通用的自定義類型映射
	// 用於將來源格式中的非標準類型轉換為標準類型
	Custom map[string]string `mapstructure:"custom"`
}

// LayoutConfig 描述 Excel 表頭各列的位置。
//
// 列號採 Excel 的寫法（第一列為 1），0 或未填表示沿用預設值：
// nameRow=2、typeRow=3、metaRow=4、outputRow=5、dataRow=6、skipMarker="skip"。
type LayoutConfig struct {
	NameRow    int    `mapstructure:"nameRow"`    // 欄位名稱列
	TypeRow    int    `mapstructure:"typeRow"`    // 值域類型列
	MetaRow    int    `mapstructure:"metaRow"`    // metadata（ref: 參考表達式）列
	OutputRow  int    `mapstructure:"outputRow"`  // 輸出類型列（both/client/skip...）
	DataRow    int    `mapstructure:"dataRow"`    // 第一筆資料所在列
	SkipMarker string `mapstructure:"skipMarker"` // 輸出類型列中代表略過的標記
}

// ToLayout 將設定轉成 source.Layout（0-based 列索引），並檢查設定是否合理。
func (c LayoutConfig) ToLayout() (source.Layout, error) {
	layout := source.DefaultLayout()

	apply := func(dst *int, excelRow int) {
		if excelRow > 0 {
			*dst = excelRow - 1
		}
	}
	apply(&layout.NameRow, c.NameRow)
	apply(&layout.TypeRow, c.TypeRow)
	apply(&layout.MetaRow, c.MetaRow)
	apply(&layout.OutputRow, c.OutputRow)
	apply(&layout.DataRow, c.DataRow)
	if strings.TrimSpace(c.SkipMarker) != "" {
		layout.SkipMarker = strings.TrimSpace(c.SkipMarker)
	}

	if err := layout.Validate(); err != nil {
		return source.Layout{}, fmt.Errorf("invalid layout config: %w", err)
	}
	return layout, nil
}

var globalConfig *Config

// Init 初始化配置系統
// 優先順序：環境變數 > 配置檔案
func Init(cfgFile string) error {
	v := viper.New()

	// 設定配置檔名稱和路徑
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// 預設搜尋路徑：formgen.yaml / formgen.yml（以及 viper 支援的其他副檔名）
		// 刻意不呼叫 SetConfigType：設定後 viper 也會比對沒有副檔名的 "formgen"，
		// 在專案根目錄執行 ./formgen 時會把執行檔本身當成設定檔讀取。
		v.SetConfigName("formgen")
		v.AddConfigPath(".")            // 當前目錄
		v.AddConfigPath("$HOME")        // 使用者目錄
		v.AddConfigPath("/etc/formgen") // 系統目錄
	}

	// 環境變數設定
	v.SetEnvPrefix("FORMGEN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 讀取配置檔（如果存在）
	if err := v.ReadInConfig(); err != nil {
		// 配置檔不存在不是錯誤，返回空配置
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析到結構體
	globalConfig = &Config{}
	if err := v.Unmarshal(globalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 確保 map 不為 nil
	if globalConfig.Mappings.Custom == nil {
		globalConfig.Mappings.Custom = make(map[string]string)
	}

	return nil
}

// Get 取得全域配置
// 如果尚未初始化，會自動初始化
func Get() *Config {
	if globalConfig == nil {
		// 自動初始化
		if err := Init(""); err != nil {
			// 初始化失敗時返回空配置
			return &Config{
				Mappings: MappingsConfig{
					Custom: map[string]string{},
				},
			}
		}
	}
	return globalConfig
}

// GetCustomMappings 取得自定義類型映射
func GetCustomMappings() map[string]string {
	return Get().Mappings.Custom
}

// GetLayout 取得表頭版面設定（已轉為 source.Layout）
func GetLayout() (source.Layout, error) {
	return Get().Layout.ToLayout()
}
