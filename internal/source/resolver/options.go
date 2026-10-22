package resolver

import "fmt"

// options 定義 Resolver 的配置選項
type options struct {
	// warnHandler 用於處理警告訊息（如找不到參考表單）
	warnHandler func(msg string)
}

// Option 定義選項函數類型
type Option func(*options)

// WithWarnHandler 設定警告處理函數
// 當找不到參考表單時會調用此函數
func WithWarnHandler(handler func(msg string)) Option {
	return func(o *options) {
		o.warnHandler = handler
	}
}

// defaultOptions 返回預設選項
func defaultOptions() *options {
	return &options{
		warnHandler: func(msg string) {
			fmt.Printf("Warning: %s\n", msg)
		},
	}
}
