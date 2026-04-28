package model

import "time"

type Member struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type DailyRecord struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	MemberID      uint      `gorm:"index" json:"member_id"`
	Member        Member    `json:"member,omitempty"`
	Date          time.Time `gorm:"index" json:"date"`
	MilitaryMerit int64     `json:"military_merit"` // 武勋
	Prosperity    int64     `json:"prosperity"`     // 繁荣
}
