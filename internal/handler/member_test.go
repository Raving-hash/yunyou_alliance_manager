package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"yunyoumanager/internal/handler"
	"yunyoumanager/internal/model"
	"yunyoumanager/internal/repository"
	"yunyoumanager/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newMemberRouter(t *testing.T) (*gin.Engine, *repository.Repository) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	r := gin.New()
	h := handler.NewMemberHandler(repo)
	r.GET("/api/members", h.List)
	r.GET("/api/members/overview", h.Overview)
	return r, repo
}

func seedRecord(t *testing.T, repo *repository.Repository, username string, dateStr string, military, prosperity int64) {
	t.Helper()
	id, err := repo.UpsertMember(username)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := time.Parse("2006-01-02", dateStr)
	err = repo.UpsertDailyRecord(model.DailyRecord{
		MemberID: id, Date: d, MilitaryMerit: military, Prosperity: prosperity,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ── GET /api/members ─────────────────────────────────────────────────────────

func TestMemberList_Empty(t *testing.T) {
	r, _ := newMemberRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var members []map[string]any
	json.Unmarshal(w.Body.Bytes(), &members)
	if len(members) != 0 {
		t.Errorf("expected empty list, got %d", len(members))
	}
}

func TestMemberList_ReturnsAll(t *testing.T) {
	r, repo := newMemberRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)
	seedRecord(t, repo, "Bob", "2024-01-01", 2000, 800)

	req := httptest.NewRequest(http.MethodGet, "/api/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var members []map[string]any
	json.Unmarshal(w.Body.Bytes(), &members)
	if len(members) != 2 {
		t.Errorf("want 2 members, got %d", len(members))
	}
}

// ── GET /api/members/overview ─────────────────────────────────────────────────

func TestOverview_ReturnsAllWithNoSearch(t *testing.T) {
	r, repo := newMemberRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)
	seedRecord(t, repo, "Bob", "2024-01-01", 2000, 800)

	req := httptest.NewRequest(http.MethodGet, "/api/members/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	members := resp["members"].([]any)
	if len(members) != 2 {
		t.Errorf("want 2, got %d", len(members))
	}
}

func TestOverview_SearchFiltersMembers(t *testing.T) {
	r, repo := newMemberRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)
	seedRecord(t, repo, "Bob", "2024-01-01", 2000, 800)
	seedRecord(t, repo, "Alibaba", "2024-01-01", 3000, 1000)

	req := httptest.NewRequest(http.MethodGet, "/api/members/overview?search=ali", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	members := resp["members"].([]any)
	if len(members) != 2 {
		t.Errorf("search 'ali' should return Alice + Alibaba, got %d", len(members))
	}
}

func TestOverview_AllianceAvgNotAffectedBySearch(t *testing.T) {
	r, repo := newMemberRouter(t)
	// Alice: 3 days, growth rate = (200-100)/100*100 = 100%
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)
	seedRecord(t, repo, "Alice", "2024-01-02", 1100, 500)
	seedRecord(t, repo, "Alice", "2024-01-03", 1300, 500)
	// Bob: 3 days, growth rate = (400-200)/200*100 = 100%
	seedRecord(t, repo, "Bob", "2024-01-01", 2000, 800)
	seedRecord(t, repo, "Bob", "2024-01-02", 2200, 800)
	seedRecord(t, repo, "Bob", "2024-01-03", 2600, 800)

	// Without search — get alliance avg
	req := httptest.NewRequest(http.MethodGet, "/api/members/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var fullResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &fullResp)
	avgFull := fullResp["alliance_avg_growth_rate"].(float64)

	// With search=Alice — alliance avg should be the same (computed from all members)
	req2 := httptest.NewRequest(http.MethodGet, "/api/members/overview?search=Alice", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var filteredResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &filteredResp)
	avgFiltered := filteredResp["alliance_avg_growth_rate"].(float64)

	if avgFull != avgFiltered {
		t.Errorf("alliance avg should be the same regardless of search: full=%.4f filtered=%.4f", avgFull, avgFiltered)
	}
	// Only Alice should be in results
	members := filteredResp["members"].([]any)
	if len(members) != 1 {
		t.Errorf("want 1 filtered member, got %d", len(members))
	}
}

func TestOverview_GrowthRateDiffFilled(t *testing.T) {
	r, repo := newMemberRouter(t)
	// Alice growth rate = 100%, Bob growth rate = 0%  → avg = 50%
	// Alice diff = 100-50=50, Bob diff = 0-50=-50
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)
	seedRecord(t, repo, "Alice", "2024-01-02", 1100, 500)
	seedRecord(t, repo, "Alice", "2024-01-03", 1300, 500) // delta0=200, delta1=100 → rate=100%

	seedRecord(t, repo, "Bob", "2024-01-01", 2000, 800)
	seedRecord(t, repo, "Bob", "2024-01-02", 2200, 800)
	seedRecord(t, repo, "Bob", "2024-01-03", 2400, 800) // delta0=200, delta1=200 → rate=0%

	req := httptest.NewRequest(http.MethodGet, "/api/members/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	avg := resp["alliance_avg_growth_rate"].(float64)
	if avg < 49.9 || avg > 50.1 {
		t.Errorf("alliance avg: want ~50, got %.4f", avg)
	}

	members := resp["members"].([]any)
	for _, mi := range members {
		m := mi.(map[string]any)
		if m["GrowthRateDiff"] == nil {
			t.Errorf("member %s should have GrowthRateDiff", m["Username"])
		}
	}
}

func TestOverview_NilAllianceAvgWhenNoGrowthData(t *testing.T) {
	r, repo := newMemberRouter(t)
	// Only 1 day of data — no growth rate for anyone
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)

	req := httptest.NewRequest(http.MethodGet, "/api/members/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["alliance_avg_growth_rate"] != nil {
		t.Errorf("expected nil alliance avg with no growth data, got %v", resp["alliance_avg_growth_rate"])
	}
}
