package handler

import (
	"net/http"
	"yunyoumanager/internal/repository"

	"github.com/gin-gonic/gin"
)

type ChartHandler struct {
	repo *repository.Repository
}

func NewChartHandler(repo *repository.Repository) *ChartHandler {
	return &ChartHandler{repo: repo}
}

// GetByMember handles GET /api/chart/:username
// Returns time-series for a single member including daily delta and 武勋率.
func (h *ChartHandler) GetByMember(c *gin.Context) {
	username := c.Param("username")

	member, err := h.repo.GetMemberByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "成员不存在"})
		return
	}

	records, err := h.repo.GetChartData(member.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	n := len(records)
	dates := make([]string, n)
	military := make([]int64, n)
	prosperity := make([]int64, n)
	// 每日武勋增量：今日武勋 - 昨日武勋；第一天无前一天数据置 nil
	militaryDelta := make([]*int64, n)
	// 武勋率 = 当日武勋增量 / 当日繁荣；繁荣为0时置 nil
	militaryRate := make([]*float64, n)

	for i, r := range records {
		dates[i] = r.Date.Format("2006-01-02")
		military[i] = r.MilitaryMerit
		prosperity[i] = r.Prosperity

		if i > 0 {
			delta := r.MilitaryMerit - records[i-1].MilitaryMerit
			militaryDelta[i] = &delta
			if r.Prosperity > 0 {
				rate := float64(delta) / float64(r.Prosperity)
				militaryRate[i] = &rate
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"member":         member,
		"dates":          dates,
		"military":       military,
		"prosperity":     prosperity,
		"military_delta": militaryDelta,
		"military_rate":  militaryRate,
	})
}

// GetAllianceTotals handles GET /api/chart/alliance
// Returns daily total military merit and prosperity across all members.
func (h *ChartHandler) GetAllianceTotals(c *gin.Context) {
	totals, err := h.repo.GetAllianceDailyTotals()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	n := len(totals)
	dates := make([]string, n)
	totalMilitary := make([]int64, n)
	totalProsperity := make([]int64, n)
	// 联盟每日武勋增量
	militaryDelta := make([]*int64, n)

	for i, t := range totals {
		dates[i] = t.Date.Format("2006-01-02")
		totalMilitary[i] = t.TotalMilitary
		totalProsperity[i] = t.TotalProsperity
		if i > 0 {
			delta := t.TotalMilitary - totals[i-1].TotalMilitary
			militaryDelta[i] = &delta
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"dates":           dates,
		"total_military":  totalMilitary,
		"total_prosperity": totalProsperity,
		"military_delta":  militaryDelta,
	})
}
