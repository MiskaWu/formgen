// Package sampledata 定義 formgen 範例與測試使用的合成 Excel 表格。
//
// 所有內容都是虛構的示範資料（一個小型 RPG 的分類、區域、任務、商店與排行獎勵），
// 用來覆蓋 formgen 支援的各種表頭寫法：
//   - 輸出類型 skip / client / both
//   - * 參考修飾符：整筆嵌入（ref:Table）、欄位引用（ref:Table.Field）、找不到參考表時的 fallback
//   - [] 陣列修飾符（資料以 JSON 陣列填寫）
//   - Array[index].Field 結構陣列
//   - SQL 保留字欄位（Order）、bool、float、自訂別名類型（entry、asset、tag、tinyint、shortint）
//
// testdata/*.xlsx 由本套件產生，修改表格後執行：
//
//	go generate ./internal/sampledata
//
//go:generate go run ./gen -out ../../testdata
package sampledata

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// NotesSheetName 是每個範例檔第二個工作表的名稱。
// formgen 只讀取第一個工作表，這個工作表用來示範「其他工作表會被忽略」。
const NotesSheetName = "欄位說明"

// Table 是一張範例表：Name 同時是檔名（不含副檔名）與第一個工作表名稱，
// Rows 依序為顯示名稱、欄位名稱、值域類型、metadata、輸出類型，之後是資料列。
type Table struct {
	Name string
	Rows [][]string
}

// FileName 返回此表的 xlsx 檔名
func (t Table) FileName() string {
	return t.Name + ".xlsx"
}

// NotesRows 返回第二個工作表（欄位說明）的內容，由表頭自動轉置而來
func (t Table) NotesRows() [][]string {
	rows := [][]string{{"顯示名稱", "欄位名稱", "值域類型", "metadata", "輸出類型"}}
	for col := range t.Rows[1] {
		row := make([]string, 0, 5)
		for r := 0; r < 5; r++ {
			row = append(row, cell(t.Rows, r, col))
		}
		rows = append(rows, row)
	}
	return rows
}

func cell(rows [][]string, r, c int) string {
	if r >= len(rows) || c >= len(rows[r]) {
		return ""
	}
	return rows[r][c]
}

// Tables 返回所有範例表
func Tables() []Table {
	return []Table{
		{
			Name: "Demo_Category",
			Rows: [][]string{
				{"分類Id", "開關", "圖示", "分類名稱", "啟用"},
				{"Id", "__extra", "Icon", "Name", "Enabled"},
				{"entry", "int", "asset", "string", "bool"},
				{"", "cond:bool", "", "", ""},
				{"both", "skip", "client", "both", "both"},
				{"1", "1", "icons/category/weapon.png", "武器", "true"},
				{"2", "1", "icons/category/armor.png", "防具", "true"},
				{"3", "0", "icons/category/material.png", "素材", "false"},
			},
		},
		{
			Name: "Demo_Region",
			Rows: [][]string{
				{"區域Id", "開關", "備註", "難度", "排序", "名稱文字", "說明文字", "地圖資源", "標籤", "相鄰區域", "舊標籤"},
				{"Id", "__extra", "Memo", "Difficulty", "Order", "NameText", "DescText", "MapAsset", "Tags", "Neighbors", "LegacyTag"},
				{"entry", "int", "text", "int", "int", "*entry", "*entry", "asset", "[]tag", "[]int", "tag"},
				{"", "cond:bool", "", "", "", "ref:TextInfo", "ref:TextInfo", "", "", "", ""},
				{"both", "skip", "skip", "both", "both", "client", "client", "client", "both", "both", "skip"},
				{"101", "1", "新手草原", "1", "1", "5001", "5002", "maps/meadow", `["grass","day"]`, "[102]", "old"},
				{"102", "1", "風蝕峽谷", "2", "2", "5003", "5004", "maps/canyon", `["desert"]`, "[101,103]"},
				{"103", "1", "熔岩山脈", "3", "3", "5005", "5006", "maps/volcano", "[]", "[102]"},
			},
		},
		{
			Name: "Demo_Mission",
			Rows: [][]string{
				{"任務Id", "開關", "備註", "所屬區域", "名稱文字", "體力消耗", "排序", "掉落群組", "推薦戰力", "推薦元素", "場景物件"},
				{"Id", "__extra", "Memo", "RegionId", "NameText", "Cost", "Order", "Drops", "SuggestPower", "Elements", "SceneActor"},
				{"entry", "int", "text", "*entry", "*entry", "tinyint", "int", "*[]int", "int", "[]*entry", "string"},
				{"", "cond:bool", "", "ref:Demo_Region", "ref:TextInfo", "", "", "ref:DropTable.GroupId", "", "ref:Element", ""},
				{"both", "skip", "skip", "both", "client", "both", "both", "both", "both", "skip", "client"},
				{"1001", "1", "第一關", "101", "6001", "5", "1", "9001", "100", "", "Spot_01"},
				{"1002", "1", "第二關", "101", "6002", "5", "2", "9002", "150", "", "Spot_02"},
				{"1003", "1", "第三關", "102", "6003", "8", "1", "9003", "300", "", "Spot_03"},
				{"1004", "1", "第四關", "103", "6004", "10", "1", "9004", "600", "", "Spot_04"},
			},
		},
		{
			Name: "Demo_ShopLabel",
			Rows: [][]string{
				{"標籤Id", "開關", "標籤文字", "標籤備註", "排序"},
				{"Id", "__extra", "LabelText", "LabelMemo", "LabelOrder"},
				{"entry", "int", "*entry", "string", "int"},
				{"", "cond:bool", "ref:TextInfo", "", ""},
				{"both", "skip", "client", "skip", "client"},
				{"1", "1", "7001", "推薦", "1"},
				{"2", "1", "7002", "日用品", "2"},
				{"3", "1", "7003", "限時", "3"},
			},
		},
		{
			Name: "Demo_ShopGoods",
			Rows: [][]string{
				{"商品Id", "開關", "企劃備註", "名稱文字", "商品群組", "獎勵類型", "獎勵Id", "權重", "數量", "支付道具", "價格", "折扣", "圖示", "限購次數"},
				{"Id", "__extra", "Note", "NameText", "GroupId", "RewardType", "RewardId", "Weight", "Amount", "PriceItem", "Price", "Discount", "Icon", "PurchaseLimit"},
				{"entry", "int", "string", "*entry", "shortint", "int", "int", "shortint", "int", "*entry", "int", "float", "asset", "int"},
				{"", "cond:bool", "", "ref:TextInfo", "", "", "", "", "", "ref:ItemInfo", "", "", "", ""},
				{"both", "skip", "skip", "client", "both", "both", "both", "both", "both", "both", "both", "both", "client", "both"},
				{"101", "1", "小型藥水", "8001", "11", "1", "2001", "10", "1", "1", "100", "1", "icons/goods/potion_s.png", "10"},
				{"102", "1", "大型藥水", "8002", "11", "1", "2002", "5", "1", "1", "250", "0.9", "icons/goods/potion_l.png", "5"},
				{"103", "1", "傳送卷軸", "8003", "12", "1", "2003", "10", "3", "1", "80", "1", "icons/goods/scroll.png", "0"},
				{"104", "1", "金幣袋", "8004", "13", "2", "1", "1", "500", "2", "30", "1", "icons/goods/coin_bag.png", "1"},
			},
		},
		{
			Name: "Demo_Shop",
			Rows: [][]string{
				{"商店Id", "開關", "標籤", "排序", "商品群組", "版型", "抽選數量", "名稱文字", "名稱備註"},
				{"Id", "__extra", "Label", "SortOrder", "GoodsGroupId", "DisplayType", "PickCount", "NameText", "NameMemo"},
				{"entry", "int", "*entry", "int", "*[]shortint", "shortint", "shortint", "*entry", "string"},
				{"", "cond:bool", "ref:Demo_ShopLabel", "", "ref:Demo_ShopGoods.GroupId", "", "", "ref:TextInfo", ""},
				{"both", "skip", "both", "client", "both", "both", "both", "client", "skip"},
				{"1", "1", "1", "1", "11", "3", "", "7101", "一般商店"},
				{"2", "1", "2", "2", "12", "3", "", "7102", "雜貨店"},
				{"3", "1", "3", "3", "13", "1", "1", "7103", "限時商店"},
			},
		},
		{
			Name: "Demo_RankReward",
			Rows: [][]string{
				{"獎勵Id", "開關", "群組", "最高名次", "最低名次", "說明文字", "道具類型", "道具Id", "數量", "道具類型", "道具Id", "數量", "道具類型", "道具Id", "數量", "信件範本"},
				{"Id", "__extra", "GroupId", "RankMax", "RankMin", "DescText", "Loot[0].Kind", "Loot[0].ItemId", "Loot[0].Amount", "Loot[1].Kind", "Loot[1].ItemId", "Loot[1].Amount", "Loot[2].Kind", "Loot[2].ItemId", "Loot[2].Amount", "MailTemplate"},
				{"entry", "int", "int", "int", "int", "int", "int", "int", "int", "int", "int", "int", "int", "int", "int", "int"},
				{"", "cond:bool", "", "", "", "ref:TextInfo", "", "", "", "", "", "", "", "", "", "ref:MailTemplate"},
				{"both", "skip", "both", "both", "both", "client", "both", "both", "both", "both", "both", "both", "both", "both", "both", "both"},
				{"1", "1", "1", "1", "1", "9101", "1", "2001", "5", "2", "1", "1000", "", "", "", "301"},
				{"2", "1", "1", "2", "3", "9101", "1", "2001", "3", "2", "1", "500", "", "", "", "301"},
				{"3", "1", "1", "4", "10", "9101", "2", "1", "200", "", "", "", "", "", "", "301"},
				{"4", "1", "2", "1", "1", "9102", "1", "2002", "2", "1", "2003", "5", "2", "1", "2000", "302"},
			},
		},
	}
}

// Build 把範例表建成 Excel 檔：第一個工作表是表格本身，第二個工作表是欄位說明。
// 空字串的儲存格不寫入，讓讀回的內容與 Rows 一致（去掉列尾空白後）。
func Build(t Table) (*excelize.File, error) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", t.Name); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}
	if err := writeRows(f, t.Name, t.Rows); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(NotesSheetName); err != nil {
		return nil, fmt.Errorf("create notes sheet: %w", err)
	}
	if err := writeRows(f, NotesSheetName, t.NotesRows()); err != nil {
		return nil, err
	}
	f.SetActiveSheet(0)
	// 固定文件屬性，讓產生結果可重現
	if err := f.SetDocProps(&excelize.DocProperties{
		Creator:        "formgen sampledata",
		Created:        "2026-01-01T00:00:00Z",
		Modified:       "2026-01-01T00:00:00Z",
		LastModifiedBy: "formgen sampledata",
	}); err != nil {
		return nil, fmt.Errorf("set doc props: %w", err)
	}
	return f, nil
}

func writeRows(f *excelize.File, sheet string, rows [][]string) error {
	for r, row := range rows {
		for c, value := range row {
			if value == "" {
				continue
			}
			name, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return err
			}
			if err := f.SetCellStr(sheet, name, value); err != nil {
				return fmt.Errorf("%s!%s: %w", sheet, name, err)
			}
		}
	}
	return nil
}

// WriteAll 把所有範例表寫到 dir，返回寫出的檔案路徑
func WriteAll(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var paths []string
	for _, t := range Tables() {
		f, err := Build(t)
		if err != nil {
			return nil, fmt.Errorf("build %s: %w", t.Name, err)
		}
		path := filepath.Join(dir, t.FileName())
		err = f.SaveAs(path)
		closeErr := f.Close()
		if err != nil {
			return nil, fmt.Errorf("save %s: %w", path, err)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if err := os.Chmod(path, 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
