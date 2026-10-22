package cmd

import (
	"fmt"
	"path/filepath"
)

// expandGlobPatterns 展開 glob 模式並返回檔案列表。
// 如果模式沒有匹配到任何檔案，會將模式本身作為路徑返回（讓後續讀取時報錯）。
func expandGlobPatterns(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid file pattern %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			// 如果沒有匹配，視為直接的檔案路徑
			files = append(files, pattern)
		} else {
			files = append(files, matches...)
		}
	}
	return files, nil
}

// defaultOutputPath 根據輸入檔案列表產生預設的輸出路徑。
// 單檔案時使用原檔名並替換副檔名，多檔案時使用 defaultName。
func defaultOutputPath(files []string, ext string, defaultName string) string {
	if len(files) == 1 {
		base := filepath.Base(files[0])
		oldExt := filepath.Ext(base)
		name := base[:len(base)-len(oldExt)]
		dir := filepath.Dir(files[0])
		return filepath.Join(dir, name+ext)
	}
	return defaultName
}
