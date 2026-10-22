package importer

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/miskawu/formgen/internal/models"
	"github.com/miskawu/formgen/internal/strcase"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// entitySchema 儲存 Entity 的 schema 資訊
type entitySchema struct {
	name    string
	columns map[string]*columnSchema // 欄位名 -> schema
}

// columnSchema 儲存欄位的 schema 資訊
type columnSchema struct {
	name          string
	isEmbedRef    bool     // 是否為完整嵌入引用（IsReference && RefField=="" && NestedColumns 有值）
	isFieldRef    bool     // 是否為字段引用（IsReference && RefField!=""）
	refEntity     string   // 引用的 Entity 名稱
	refField      string   // 引用的字段名稱（用於字段引用）
	nestedColumns []string // 嵌套欄位名稱（用於序列化）
}

// tableData 儲存表的數據索引
type tableData struct {
	rowsByID    map[string]map[string]any   // ID -> 該行的所有欄位數據
	rowsByField map[string][]map[string]any // 字段值 -> 匹配的行數據列表（用於字段引用）
}

// EnsureSnapshotTables 確保 snapshot/version 表存在
// 應在事務外調用（DDL 不應該在事務內執行）
func (s *Importer) EnsureSnapshotTables() error {
	if err := s.db.AutoMigrate(&models.Snapshot{}, &models.Version{}); err != nil {
		return fmt.Errorf("failed to auto migrate snapshot/version tables: %w", err)
	}
	return nil
}

// Import 從已載入的 Sheet 執行導入（使用內部事務）
func (s *Importer) Import(sheet Sheet) error {
	// 如果啟用快照功能，確保表已創建
	if s.enableSnapshot {
		if err := s.EnsureSnapshotTables(); err != nil {
			return err
		}
	}

	// 使用事務處理所有插入操作
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.ImportWithTx(tx, sheet)
	})
}

// ImportWithTx 從已載入的 Sheet 執行導入（使用外部傳入的事務）
// 這允許多個 sheet 在同一事務中處理，確保原子性
func (s *Importer) ImportWithTx(tx *gorm.DB, sheet Sheet) error {
	tableName := formatTableName(sheet.Name)

	// 如果是 truncate 模式，先清空表
	if s.importMode == ImportModeTruncate {
		if err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s", tableName)).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", tableName, err)
		}
	}
	if err := s.importSheet(tx, sheet, tableName); err != nil {
		return fmt.Errorf("failed to import table %s: %w", tableName, err)
	}

	return nil
}

// importSheet 將單個工作表的數據導入資料庫
func (s *Importer) importSheet(tx *gorm.DB, sheet Sheet, tableName string) error {
	rows := sheet.Rows
	layout := s.layout
	if err := layout.Validate(); err != nil {
		return err
	}
	if len(rows) < layout.DataRow+1 {
		return fmt.Errorf("invalid sheet data: need at least %d rows (header + data)", layout.DataRow+1)
	}

	// 表頭位置由 layout 決定（預設：row 1 欄位名稱、row 4 輸出類型、row 5+ 數據行）
	fieldNames := rows[layout.NameRow]

	// 構建有效欄位索引映射
	// validFieldNames 存儲轉換後的 snake_case 欄位名（用於 DB 操作，會處理保留關鍵字）
	// protoFieldNames 存儲 snake_case 欄位名（用於 protobuf snapshot，與 proto 生成器一致）
	// originalFieldNames 存儲原始欄位名（用於查找 schema 中的嵌入資訊）
	var validFieldIndices []int
	var validFieldNames []string
	var protoFieldNames []string
	var originalFieldNames []string

	// 結構陣列欄位映射: colIdx -> arrayFieldInfo
	structFieldMap := make(map[int]arrayFieldInfo)
	// 結構陣列欄位的 colIdx（依欄位順序），確保聚合出的 JSON 欄位順序穩定
	var structFieldCols []int

	for colIdx, fieldName := range fieldNames {
		fieldName = strings.TrimSpace(fieldName)
		if fieldName == "" {
			continue
		}
		// 檢查是否需要跳過
		if layout.IsSkipped(rows, colIdx) {
			continue
		}

		// 檢查是否為結構陣列欄位（如 Loot[0].ItemId）
		if arrayName, index, nestedField, isStruct := parseStructField(fieldName); isStruct {
			structFieldMap[colIdx] = arrayFieldInfo{
				arrayName: arrayName,
				index:     index,
				fieldName: nestedField,
			}
			structFieldCols = append(structFieldCols, colIdx)
			continue // 不加入普通欄位列表
		}

		validFieldIndices = append(validFieldIndices, colIdx)
		validFieldNames = append(validFieldNames, s.formatColumnName(fieldName))
		// Proto 欄位名使用 snake_case（與 proto 生成器的 formatFieldName 一致）
		protoFieldNames = append(protoFieldNames, strcase.ToSnakeCase(fieldName))
		// 原始欄位名用於查找 schema 中的嵌入資訊
		originalFieldNames = append(originalFieldNames, fieldName)
	}

	// 處理數據行
	dataRows := rows[layout.DataRow:]
	totalRows := len(dataRows)

	// 收集所有行數據用於快照
	var snapshot []map[string]any
	if s.enableSnapshot && s.protoSerializer != nil {
		snapshot = make([]map[string]any, 0, totalRows)
	}

	for rowIdx, dataRow := range dataRows {
		// 創建動態數據 map
		// data 使用 snake_case 欄位名（用於 DB 操作）
		// snapshotData 使用原始欄位名（用於 protobuf snapshot）
		data := make(map[string]any)
		var snapshotData map[string]any
		if snapshot != nil {
			snapshotData = make(map[string]any)
		}

		// 創建結構陣列收集器
		collector := newStructFieldCollector()

		// 遍歷有效欄位（普通欄位）
		for i, colIdx := range validFieldIndices {
			if colIdx >= len(dataRow) {
				continue
			}

			cellValue := strings.TrimSpace(dataRow[colIdx])

			// 如果單元格為空，跳過
			if cellValue == "" {
				continue
			}

			// DB 使用 snake_case 欄位名（可能有關鍵字後綴）
			data[validFieldNames[i]] = cellValue
			// Snapshot 使用 snake_case 欄位名（與 proto 生成器一致）
			if snapshotData != nil {
				// 檢查是否為引用欄位（嵌入引用或字段引用）
				if refInfo := s.getRefColumnInfo(sheet.Name, originalFieldNames[i]); refInfo != nil {
					if refInfo.isEmbedRef {
						// 完整嵌入引用：按 ID 查找單筆數據
						embeddedData := s.lookupEmbeddedDataByID(refInfo.refEntity, cellValue)
						if embeddedData != nil {
							snapshotData[protoFieldNames[i]] = embeddedData
						} else {
							// 找不到嵌入數據，使用原始值
							snapshotData[protoFieldNames[i]] = cellValue
						}
					} else if refInfo.isFieldRef {
						// 字段引用：按字段值查找多筆數據
						embeddedDataList := s.lookupEmbeddedDataByField(refInfo.refEntity, refInfo.refField, cellValue)
						if len(embeddedDataList) > 0 {
							snapshotData[protoFieldNames[i]] = embeddedDataList
						} else {
							// 找不到嵌入數據，使用空陣列
							snapshotData[protoFieldNames[i]] = []map[string]any{}
						}
					}
				} else {
					snapshotData[protoFieldNames[i]] = cellValue
				}
			}
		}

		// 收集結構陣列欄位值（依欄位順序）
		for _, colIdx := range structFieldCols {
			info := structFieldMap[colIdx]
			if colIdx >= len(dataRow) {
				continue
			}
			cellValue := strings.TrimSpace(dataRow[colIdx])
			if cellValue == "" {
				continue
			}
			collector.Add(info.arrayName, info.index, info.fieldName, cellValue)
		}

		// 將結構陣列聚合為 JSON 並加入 data
		jsonFields, err := collector.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to aggregate struct fields in row %d: %w", rowIdx+layout.DataRow+1, err)
		}
		for arrayName, jsonStr := range jsonFields {
			dbColumnName := s.formatColumnName(arrayName)
			data[dbColumnName] = jsonStr
		}

		// 將結構陣列加入 snapshot（使用 []map[string]any 格式）
		if snapshotData != nil {
			for arrayName, sliceData := range collector.ToSlice() {
				snapshotData[strcase.ToSnakeCase(arrayName)] = sliceData
			}
		}

		// 跳過空行
		if len(data) == 0 {
			continue
		}

		// 根據導入模式執行不同的操作
		switch s.importMode {
		case ImportModeReplace:
			// REPLACE INTO (MySQL) - 先刪除再插入
			// 使用 GORM 的原生 SQL 執行 REPLACE INTO
			columns := make([]string, 0, len(data))
			placeholders := make([]string, 0, len(data))
			values := make([]any, 0, len(data))

			for col, val := range data {
				columns = append(columns, col)
				placeholders = append(placeholders, "?")
				values = append(values, val)
			}

			sql := fmt.Sprintf("REPLACE INTO %s (%s) VALUES (%s)",
				tableName,
				strings.Join(columns, ","),
				strings.Join(placeholders, ","))

			err = tx.Exec(sql, values...).Error
		case ImportModeUpsert:
			// MySQL: ON DUPLICATE KEY UPDATE；SQLite/PostgreSQL: ON CONFLICT (key) DO UPDATE
			if len(validFieldNames) == 0 {
				return fmt.Errorf("upsert mode requires at least one plain column as key")
			}
			err = tx.Table(tableName).Clauses(upsertClause(validFieldNames[0], data)).Create(data).Error
		default:
			// INSERT (默認模式和 truncate 模式)
			err = tx.Table(tableName).Create(data).Error
		}

		if err != nil {
			return fmt.Errorf("failed to insert row %d: %w", rowIdx+layout.DataRow+1, err)
		}

		// 收集數據用於後續快照（使用原始欄位名）
		if snapshotData != nil {
			snapshot = append(snapshot, snapshotData)
		}

		// 回報進度
		if s.onProgress != nil && (rowIdx+1)%s.batchSize == 0 {
			s.onProgress(tableName, rowIdx+1, totalRows)
		}
	}

	// 最終進度回報
	if s.onProgress != nil {
		s.onProgress(tableName, totalRows, totalRows)
	}

	// 如果啟用快照功能，使用包裝 message 保存所有數據為一份快照
	// 被嵌入的 Entity 不需要單獨生成 snapshot（數據已嵌入到父 Entity 的 protobuf 中）
	if len(snapshot) > 0 {
		if s.embeddedEntities != nil && s.embeddedEntities[sheet.Name] {
			fmt.Printf("  ⏭️  跳過 %s 的 snapshot（被嵌入的 Entity）\n", sheet.Name)
		} else {
			checksumChanged, err := s.saveSnapshot(tx, sheet.Name, snapshot)
			if err != nil {
				return fmt.Errorf("failed to save snapshot for %s: %w", sheet.Name, err)
			}

			// 如果 checksum 有變化或是第一次寫入，保存新版本
			if checksumChanged {
				if err := s.saveVersion(tx, sheet.Name); err != nil {
					return fmt.Errorf("failed to save version for %s: %w", sheet.Name, err)
				}
			}
		}
	}

	return nil
}

// saveSnapshot 使用包裝 message 保存所有數據為一份快照
// 採用 copy-on-write 策略：checksum 相同則跳過，變更時才更新並產生新版本
// 回傳 checksumChanged 表示 checksum 是否有變化或是第一次寫入
func (s *Importer) saveSnapshot(tx *gorm.DB, tableName string, dataList []map[string]any) (checksumChanged bool, err error) {
	// 序列化數據
	// wrapperName 和 itemName 都需要使用與 Proto 生成器一致的命名格式
	itemName := formatMessageName(tableName)
	protoBytes, err := s.protoSerializer.Serialize(wrapperName(tableName), itemName, dataList)
	if err != nil {
		return false, fmt.Errorf("failed to serialize data list: %w", err)
	}

	// Gzip 壓縮
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(protoBytes); err != nil {
		return false, fmt.Errorf("failed to gzip compress: %w", err)
	}
	if err := gw.Close(); err != nil {
		return false, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	compressedBytes := buf.Bytes()

	// 計算 checksum (MD5 of compressed bytes)
	checksum := calculateChecksum(compressedBytes)

	// 查詢現有的 snapshot
	var existingSnapshot models.Snapshot
	result := tx.Where("namespace = ? AND format = ? AND name = ?",
		s.namespace, "protobuf+gzip", tableName).
		First(&existingSnapshot)

	if result.Error == nil && existingSnapshot.Checksum == checksum {
		// checksum 相同，不做任何變更
		return false, nil
	}

	// 準備新的快照記錄
	snapshot := models.Snapshot{
		Namespace: s.namespace,
		Format:    "protobuf+gzip",
		Name:      tableName,
		Checksum:  checksum,
		Data:      compressedBytes,
	}

	if result.Error == gorm.ErrRecordNotFound {
		// 第一次寫入，創建新記錄
		if err := tx.Create(&snapshot).Error; err != nil {
			return false, fmt.Errorf("failed to insert snapshot: %w", err)
		}
	} else {
		// checksum 變化，更新現有記錄
		if err := tx.Model(&existingSnapshot).Updates(map[string]any{
			"checksum": checksum,
			"data":     compressedBytes,
		}).Error; err != nil {
			return false, fmt.Errorf("failed to update snapshot: %w", err)
		}
	}

	return true, nil
}

// upsertClause 建立 upsert 用的 OnConflict 子句。
//
// 匯入的資料是 map、沒有 GORM schema，OnConflict{UpdateAll: true} 無法推導要更新的欄位
// （會得到 "model value required"），因此明確列出衝突鍵與要更新的欄位。
// 衝突鍵取第一個一般欄位，與 SQL 生成器把第一欄設為 PRIMARY KEY 的慣例一致；
// MySQL 的 ON DUPLICATE KEY UPDATE 會忽略 Columns，直接依主鍵／唯一鍵判斷。
func upsertClause(keyColumn string, data map[string]any) clause.OnConflict {
	columns := make([]string, 0, len(data))
	for col := range data {
		if col != keyColumn {
			columns = append(columns, col)
		}
	}
	sort.Strings(columns)

	onConflict := clause.OnConflict{Columns: []clause.Column{{Name: keyColumn}}}
	if len(columns) == 0 {
		onConflict.DoNothing = true
	} else {
		onConflict.DoUpdates = clause.AssignmentColumns(columns)
	}
	return onConflict
}

// wrapperName 根據表名生成包裝 message 名稱
// 與 Proto 生成器的命名規則保持一致：PascalCase + 移除底線 + "List" 後綴
// 例如: Demo_Category -> DemoCategoryList
func wrapperName(tableName string) string {
	return formatMessageName(tableName) + "List"
}

// formatMessageName 將名稱轉為 PascalCase（移除底線）
// 與 Proto 生成器的 formatMessageName 保持一致
func formatMessageName(name string) string {
	return strings.ReplaceAll(strcase.ToPascalCase(name), "_", "")
}

// calculateChecksum 計算 protobuf bytes 的 MD5 checksum
func calculateChecksum(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// saveVersion 保存新的版本記錄
func (s *Importer) saveVersion(tx *gorm.DB, name string) error {
	version := models.Version{
		Namespace: s.namespace,
		Name:      name,
		Version:   uuid.New().String(),
	}

	if err := tx.Create(&version).Error; err != nil {
		return fmt.Errorf("failed to insert version: %w", err)
	}

	return nil
}

// formatTableName 將表名轉為 snake_case（與 SQL 生成器保持一致）
func formatTableName(name string) string {
	return strcase.ToSnakeCase(name)
}

// formatColumnName 將欄位名轉為 snake_case 並處理保留關鍵字（與 SQL 生成器保持一致）
func (s *Importer) formatColumnName(name string) string {
	snakeName := strcase.ToSnakeCase(name)
	if strcase.IsSQLReservedKeyword(snakeName) {
		return snakeName + s.keywordSuffix
	}
	return snakeName
}

// IndexSheetData 建立 sheet 數據的索引（用於嵌入數據查找）
// 應在 ImportWithTx 之前調用，先索引所有被嵌入的表
// 支援兩種索引：
//   - rowsByID: 按 ID 索引（用於 isEmbedRef）
//   - rowsByField: 按指定字段值索引（用於 isFieldRef）
func (s *Importer) IndexSheetData(sheet Sheet) error {
	if s.dataIndex == nil {
		return nil // 未啟用嵌入功能
	}

	rows := sheet.Rows
	layout := s.layout
	if err := layout.Validate(); err != nil {
		return err
	}
	if len(rows) < layout.DataRow+1 {
		return fmt.Errorf("invalid sheet data: need at least %d rows", layout.DataRow+1)
	}

	fieldNames := rows[layout.NameRow]
	dataRows := rows[layout.DataRow:]

	// 收集需要被字段索引的欄位（從其他 Entity 的 RefField 來判斷）
	fieldIndexNames := s.getFieldIndexNames(sheet.Name)

	// 找出 ID 欄位的索引和有效欄位
	idColIdx := -1
	fieldIndexColMap := make(map[string]int) // fieldName -> colIdx（用於字段索引）
	var validColIndices []int
	var validFieldNames []string

	for colIdx, fieldName := range fieldNames {
		fieldName = strings.TrimSpace(fieldName)
		if fieldName == "" {
			continue
		}
		if layout.IsSkipped(rows, colIdx) {
			continue
		}

		// 記錄 ID 欄位索引（不區分大小寫）
		if strings.EqualFold(fieldName, "id") {
			idColIdx = colIdx
		}

		// 記錄需要被字段索引的欄位
		if fieldIndexNames[fieldName] {
			fieldIndexColMap[fieldName] = colIdx
		}

		validColIndices = append(validColIndices, colIdx)
		validFieldNames = append(validFieldNames, fieldName)
	}

	// 建立數據索引
	td := &tableData{
		rowsByID:    make(map[string]map[string]any),
		rowsByField: make(map[string][]map[string]any),
	}

	for _, dataRow := range dataRows {
		// 建立行數據
		rowData := make(map[string]any)
		for i, colIdx := range validColIndices {
			if colIdx >= len(dataRow) {
				continue
			}
			cellValue := strings.TrimSpace(dataRow[colIdx])
			if cellValue != "" {
				// 使用 snake_case 欄位名（與 proto 欄位名一致）
				rowData[strcase.ToSnakeCase(validFieldNames[i])] = cellValue
			}
		}

		// 按 ID 索引
		if idColIdx != -1 && idColIdx < len(dataRow) {
			id := strings.TrimSpace(dataRow[idColIdx])
			if id != "" {
				td.rowsByID[id] = rowData
			}
		}

		// 按字段值索引
		for fieldName, colIdx := range fieldIndexColMap {
			if colIdx >= len(dataRow) {
				continue
			}
			fieldValue := strings.TrimSpace(dataRow[colIdx])
			if fieldValue != "" {
				// 使用 "fieldName:value" 作為 key，方便區分不同字段
				key := fieldName + ":" + fieldValue
				td.rowsByField[key] = append(td.rowsByField[key], rowData)
			}
		}
	}

	s.dataIndex[sheet.Name] = td
	return nil
}

// getFieldIndexNames 獲取需要被字段索引的欄位名稱（來自其他 Entity 的 RefField）
func (s *Importer) getFieldIndexNames(sheetName string) map[string]bool {
	result := make(map[string]bool)

	if s.entityIndex == nil {
		return result
	}

	// 遍歷所有 Entity，找出引用此 sheet 的字段引用
	for _, es := range s.entityIndex {
		for _, cs := range es.columns {
			if cs.isFieldRef && cs.refEntity == sheetName && cs.refField != "" {
				result[cs.refField] = true
			}
		}
	}

	return result
}

// lookupEmbeddedDataByID 根據 ID 查找被嵌入表的單筆數據（用於 isEmbedRef）
func (s *Importer) lookupEmbeddedDataByID(refEntity string, id string) map[string]any {
	if s.dataIndex == nil {
		return nil
	}

	td, ok := s.dataIndex[refEntity]
	if !ok {
		return nil
	}

	return td.rowsByID[id]
}

// lookupEmbeddedDataByField 根據字段值查找被嵌入表的多筆數據（用於 isFieldRef）
func (s *Importer) lookupEmbeddedDataByField(refEntity, refField, fieldValue string) []map[string]any {
	if s.dataIndex == nil {
		return nil
	}

	td, ok := s.dataIndex[refEntity]
	if !ok {
		return nil
	}

	key := refField + ":" + fieldValue
	return td.rowsByField[key]
}

// getRefColumnInfo 獲取欄位的引用資訊（包含 isEmbedRef 和 isFieldRef）
func (s *Importer) getRefColumnInfo(sheetName, fieldName string) *columnSchema {
	if s.entityIndex == nil {
		return nil
	}

	es, ok := s.entityIndex[sheetName]
	if !ok {
		return nil
	}

	cs, ok := es.columns[fieldName]
	if !ok {
		return nil
	}

	if !cs.isEmbedRef && !cs.isFieldRef {
		return nil
	}

	return cs
}
