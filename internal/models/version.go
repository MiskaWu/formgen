package models

import "gorm.io/gorm"

// Version 記錄某張資料表的快照每次變動，Version 欄位是隨機產生的 UUID。
type Version struct {
	Namespace string `gorm:"type:varchar(32);index:idx_version_lookup,priority:1"` // 命名空間
	Name      string `gorm:"type:varchar(32);index:idx_version_lookup,priority:2"` // 資料表（Entity）名稱
	Version   string `gorm:"type:varchar(36)"`                                     // 版本號（UUID）
	gorm.Model
}
