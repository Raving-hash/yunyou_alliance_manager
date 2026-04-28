package handler

import (
	"net/http"
	"strings"
	"yunyoumanager/internal/repository"

	"github.com/gin-gonic/gin"
)

type MemberHandler struct {
	repo *repository.Repository
}

func NewMemberHandler(repo *repository.Repository) *MemberHandler {
	return &MemberHandler{repo: repo}
}

// List handles GET /api/members
func (h *MemberHandler) List(c *gin.Context) {
	members, err := h.repo.ListMembers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}

// Overview handles GET /api/members/overview?search=xxx
func (h *MemberHandler) Overview(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))

	all, err := h.repo.GetMembersOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Compute alliance average growth rate from all members.
	var sum float64
	var count int
	for _, m := range all {
		if m.GrowthRate != nil {
			sum += *m.GrowthRate
			count++
		}
	}
	var allianceAvg *float64
	if count > 0 {
		avg := sum / float64(count)
		allianceAvg = &avg
	}

	// Filter by search (case-insensitive) and fill GrowthRateDiff.
	result := make([]repository.MemberOverview, 0, len(all))
	for i := range all {
		if search != "" && !strings.Contains(strings.ToLower(all[i].Username), strings.ToLower(search)) {
			continue
		}
		if all[i].GrowthRate != nil && allianceAvg != nil {
			diff := *all[i].GrowthRate - *allianceAvg
			all[i].GrowthRateDiff = &diff
		}
		result = append(result, all[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"alliance_avg_growth_rate": allianceAvg,
		"members":                  result,
	})
}
