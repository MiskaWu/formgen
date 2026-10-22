package excel

// loader 定義在 loader.go 中實作
type loader struct {
}

// NewLoader 建立新的 Excel Loader
func NewLoader() *loader {
	return &loader{}
}
