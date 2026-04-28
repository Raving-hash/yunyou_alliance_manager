// cmd/seed/main.go — 生成30人7日测试数据，直接写入数据库
package main

import (
	"log"
	"math/rand"
	"time"
	"yunyoumanager/internal/model"
	"yunyoumanager/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var names = []string{
	"凌霄剑客", "暗夜猎手", "烈焰狂刃", "苍穹之翼", "幽冥鬼将",
	"破军战神", "霜晨月影", "碧落仙踪", "赤焰战魂", "玄冰剑圣",
	"紫电青霜", "无双剑帝", "天煞孤星", "飞鸿踏雪", "墨染江湖",
	"沧海一粟", "金戈铁马", "踏雪无痕", "青龙偃月", "白虎神将",
	"玄武水灵", "朱雀火凤", "银河落九天", "沙场秋点兵", "醉卧沙场",
	"征战四方", "铁血柔情", "乱世枭雄", "江湖浪子", "逆天改命",
}

func main() {
	db, err := gorm.Open(sqlite.Open("data/yunyou.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	db.AutoMigrate(&model.Member{}, &model.DailyRecord{})
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_member_date ON daily_records(member_id, date)`)

	repo := repository.New(db)
	rng := rand.New(rand.NewSource(42))

	today := time.Now().Truncate(24 * time.Hour).UTC()

	for _, name := range names {
		id, err := repo.UpsertMember(name)
		if err != nil {
			log.Fatalf("创建成员 %s 失败: %v", name, err)
		}

		// 各人初始武勋 50000~300000，繁荣 5000~20000
		military := int64(50000 + rng.Intn(250000))
		prosperity := int64(5000 + rng.Intn(15000))

		for dayOffset := -6; dayOffset <= 0; dayOffset++ {
			date := today.AddDate(0, 0, dayOffset)

			// 每日武勋增量：500~8000，偶尔有人摆烂（0~200）
			var delta int64
			if rng.Intn(10) < 2 { // 20% 概率当天没打
				delta = int64(rng.Intn(200))
			} else {
				delta = int64(500 + rng.Intn(7500))
			}
			military += delta

			// 繁荣小幅波动 ±5%
			change := int64(float64(prosperity) * (rng.Float64()*0.1 - 0.05))
			prosperity += change
			if prosperity < 1000 {
				prosperity = 1000
			}

			err := repo.UpsertDailyRecord(model.DailyRecord{
				MemberID:      id,
				Date:          date,
				MilitaryMerit: military,
				Prosperity:    prosperity,
			})
			if err != nil {
				log.Fatalf("写入 %s %s 失败: %v", name, date.Format("2006-01-02"), err)
			}
		}
		log.Printf("✓ %-12s  当前武勋 %d  繁荣 %d", name, military, prosperity)
	}

	log.Println("完成：30人 × 7日数据已写入")
}
