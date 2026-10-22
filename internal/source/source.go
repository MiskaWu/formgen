package source

// Sheet 代表從各種來源載入的表格資料
type Sheet struct {
	Name string
	Rows [][]string
}

// ParserOption 用於配置 Parser
type ParserOption func(*parser)

// WithLayout 指定表頭版面；未指定時使用 DefaultLayout()。
func WithLayout(layout Layout) ParserOption {
	return func(p *parser) {
		p.layout = layout
	}
}

// NewParser 建立新的 Parser
func NewParser(opts ...ParserOption) *parser {
	p := &parser{layout: DefaultLayout()}
	for _, opt := range opts {
		opt(p)
	}
	return p
}
