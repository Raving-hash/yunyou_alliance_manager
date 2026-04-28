package main

import (
	"log"
	"os"
	"yunyoumanager/config"
	"yunyoumanager/internal/model"
	"yunyoumanager/internal/repository"
	"yunyoumanager/internal/router"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Ensure data directory exists.
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("创建data目录失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(config.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// Auto-migrate schema.
	if err := db.AutoMigrate(&model.Member{}, &model.DailyRecord{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// Add unique constraint for (member_id, date) if not exists.
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_member_date ON daily_records(member_id, date)`)

	repo := repository.New(db)
	r := router.Setup(repo)

	log.Printf("服务启动，监听 %s", config.Port)
	if err := r.Run(config.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
