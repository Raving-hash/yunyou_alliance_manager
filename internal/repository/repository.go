package repository

import (
	"math"
	"time"
	"yunyoumanager/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// UpsertMember finds or creates a member by username, returns its ID.
func (r *Repository) UpsertMember(username string) (uint, error) {
	m := model.Member{Username: username}
	result := r.db.Where(model.Member{Username: username}).FirstOrCreate(&m)
	return m.ID, result.Error
}

// UpsertDailyRecord inserts or updates a record for (memberID, date).
func (r *Repository) UpsertDailyRecord(rec model.DailyRecord) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "member_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"military_merit", "prosperity"}),
	}).Create(&rec).Error
}

// ListMembers returns all members ordered by username.
func (r *Repository) ListMembers() ([]model.Member, error) {
	var members []model.Member
	err := r.db.Order("username").Find(&members).Error
	return members, err
}

// GetChartData returns daily records for a member ordered by date.
func (r *Repository) GetChartData(memberID uint) ([]model.DailyRecord, error) {
	var records []model.DailyRecord
	err := r.db.Where("member_id = ?", memberID).Order("date").Find(&records).Error
	return records, err
}

// GetAllChartData returns all daily records ordered by date then member.
func (r *Repository) GetAllChartData() ([]model.DailyRecord, error) {
	var records []model.DailyRecord
	err := r.db.Preload("Member").Order("date, member_id").Find(&records).Error
	return records, err
}

// GetMemberByUsername returns a member by username.
func (r *Repository) GetMemberByUsername(username string) (model.Member, error) {
	var m model.Member
	err := r.db.Where("username = ?", username).First(&m).Error
	return m, err
}

// GetChartDataByUsername returns daily records for a member by username, ordered by date.
func (r *Repository) GetChartDataByUsername(username string) ([]model.DailyRecord, error) {
	var m model.Member
	if err := r.db.Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return r.GetChartData(m.ID)
}

// MemberOverview holds a member's latest stats and recent 3-day records.
type MemberOverview struct {
	ID                uint
	Username          string
	CurrentMilitary   int64
	CurrentProsperity int64
	// GrowthRate = (latestDelta - prevDelta) / |prevDelta| * 100, nil if insufficient data
	GrowthRate     *float64
	// GrowthRateDiff = GrowthRate - allianceAvg, filled by handler
	GrowthRateDiff *float64
	Recent         []RecentRecord
}

type RecentRecord struct {
	Date          string
	Military      int64
	Prosperity    int64
	MilitaryDelta *int64
}

// GetMembersOverview returns all members with latest values, last 3 days, and growth rate.
// Always fetches all members; caller filters by search if needed.
func (r *Repository) GetMembersOverview() ([]MemberOverview, error) {
	var records []model.DailyRecord
	if err := r.db.Preload("Member").Order("member_id, date desc").Find(&records).Error; err != nil {
		return nil, err
	}

	// Group: keep at most 3 records per member (already desc by date).
	grouped := map[uint][]model.DailyRecord{}
	order := []uint{}
	for _, rec := range records {
		if _, ok := grouped[rec.MemberID]; !ok {
			order = append(order, rec.MemberID)
		}
		if len(grouped[rec.MemberID]) < 3 {
			grouped[rec.MemberID] = append(grouped[rec.MemberID], rec)
		}
	}

	result := make([]MemberOverview, 0, len(order))
	for _, mid := range order {
		recs := grouped[mid]
		if len(recs) == 0 {
			continue
		}
		ov := MemberOverview{
			ID:                recs[0].MemberID,
			Username:          recs[0].Member.Username,
			CurrentMilitary:   recs[0].MilitaryMerit,
			CurrentProsperity: recs[0].Prosperity,
		}
		for i, rec := range recs {
			rr := RecentRecord{
				Date:       rec.Date.Format("2006-01-02"),
				Military:   rec.MilitaryMerit,
				Prosperity: rec.Prosperity,
			}
			if i+1 < len(recs) {
				delta := rec.MilitaryMerit - recs[i+1].MilitaryMerit
				rr.MilitaryDelta = &delta
			}
			ov.Recent = append(ov.Recent, rr)
		}

		// GrowthRate needs 3 days: delta0 = recs[0]-recs[1], delta1 = recs[1]-recs[2]
		if len(recs) >= 3 {
			delta0 := recs[0].MilitaryMerit - recs[1].MilitaryMerit
			delta1 := recs[1].MilitaryMerit - recs[2].MilitaryMerit
			if delta1 != 0 {
				rate := float64(delta0-delta1) / math.Abs(float64(delta1)) * 100
				ov.GrowthRate = &rate
			}
		}

		result = append(result, ov)
	}
	return result, nil
}

// DailyTotal holds aggregated per-day military merit and prosperity totals.
type DailyTotal struct {
	Date          time.Time
	TotalMilitary int64
	TotalProsperity int64
}

// GetAllianceDailyTotals returns sum of military_merit and prosperity per day.
func (r *Repository) GetAllianceDailyTotals() ([]DailyTotal, error) {
	var rows []DailyTotal
	err := r.db.Model(&model.DailyRecord{}).
		Select("date, SUM(military_merit) AS total_military, SUM(prosperity) AS total_prosperity").
		Group("date").
		Order("date").
		Scan(&rows).Error
	return rows, err
}

// TruncateDate strips time component, keeping only the date at midnight UTC.
func TruncateDate(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, time.UTC)
}
