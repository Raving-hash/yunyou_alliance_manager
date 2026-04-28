package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"yunyoumanager/internal/handler"
	"yunyoumanager/internal/repository"
	"yunyoumanager/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newChartRouter(t *testing.T) (*gin.Engine, *repository.Repository) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	r := gin.New()
	h := handler.NewChartHandler(repo)
	r.GET("/api/chart/alliance", h.GetAllianceTotals)
	r.GET("/api/chart/:username", h.GetByMember)
	return r, repo
}

// ── GET /api/chart/:username ──────────────────────────────────────────────────

func TestChartByMember_NotFound(t *testing.T) {
	r, _ := newChartRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/chart/Ghost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestChartByMember_SingleDay_NoDelta(t *testing.T) {
	r, repo := newChartRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)

	req := httptest.NewRequest(http.MethodGet, "/api/chart/Alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	dates := resp["dates"].([]any)
	if len(dates) != 1 {
		t.Errorf("want 1 date, got %d", len(dates))
	}
	military := resp["military"].([]any)
	if military[0].(float64) != 1000 {
		t.Errorf("military: want 1000, got %v", military[0])
	}
	// First element must have nil delta and nil rate
	deltas := resp["military_delta"].([]any)
	if deltas[0] != nil {
		t.Errorf("first delta should be nil, got %v", deltas[0])
	}
	rates := resp["military_rate"].([]any)
	if rates[0] != nil {
		t.Errorf("first rate should be nil, got %v", rates[0])
	}
}

func TestChartByMember_MultiDay_DeltaAndRate(t *testing.T) {
	r, repo := newChartRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)
	seedRecord(t, repo, "Alice", "2024-01-02", 1300, 400)

	req := httptest.NewRequest(http.MethodGet, "/api/chart/Alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	deltas := resp["military_delta"].([]any)
	// day2 delta = 1300-1000 = 300
	if deltas[1].(float64) != 300 {
		t.Errorf("delta[1]: want 300, got %v", deltas[1])
	}

	rates := resp["military_rate"].([]any)
	// rate = 300/400 = 0.75
	got := rates[1].(float64)
	if got < 0.749 || got > 0.751 {
		t.Errorf("rate[1]: want 0.75, got %.4f", got)
	}
}

func TestChartByMember_ZeroProsperity_NilRate(t *testing.T) {
	r, repo := newChartRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 0)
	seedRecord(t, repo, "Alice", "2024-01-02", 1200, 0)

	req := httptest.NewRequest(http.MethodGet, "/api/chart/Alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	rates := resp["military_rate"].([]any)
	// prosperity=0 → rate must be nil even when delta exists
	if rates[1] != nil {
		t.Errorf("rate should be nil when prosperity=0, got %v", rates[1])
	}
}

func TestChartByMember_MemberUsernameInResponse(t *testing.T) {
	r, repo := newChartRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 500)

	req := httptest.NewRequest(http.MethodGet, "/api/chart/Alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	member := resp["member"].(map[string]any)
	if member["username"] != "Alice" {
		t.Errorf("want username=Alice, got %v", member["username"])
	}
}

// ── GET /api/chart/alliance ───────────────────────────────────────────────────

func TestAllianceChart_Empty(t *testing.T) {
	r, _ := newChartRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/chart/alliance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	dates := resp["dates"].([]any)
	if len(dates) != 0 {
		t.Errorf("expected empty dates, got %d", len(dates))
	}
}

func TestAllianceChart_SumsAcrossMembers(t *testing.T) {
	r, repo := newChartRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 100)
	seedRecord(t, repo, "Bob", "2024-01-01", 500, 50)
	seedRecord(t, repo, "Alice", "2024-01-02", 1200, 120)
	seedRecord(t, repo, "Bob", "2024-01-02", 700, 70)

	req := httptest.NewRequest(http.MethodGet, "/api/chart/alliance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	totalMil := resp["total_military"].([]any)
	if totalMil[0].(float64) != 1500 {
		t.Errorf("day1 total: want 1500, got %v", totalMil[0])
	}
	if totalMil[1].(float64) != 1900 {
		t.Errorf("day2 total: want 1900, got %v", totalMil[1])
	}
}

func TestAllianceChart_FirstDayNilDelta(t *testing.T) {
	r, repo := newChartRouter(t)
	seedRecord(t, repo, "Alice", "2024-01-01", 1000, 100)
	seedRecord(t, repo, "Alice", "2024-01-02", 1200, 120)

	req := httptest.NewRequest(http.MethodGet, "/api/chart/alliance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	deltas := resp["military_delta"].([]any)
	if deltas[0] != nil {
		t.Errorf("first alliance delta should be nil, got %v", deltas[0])
	}
	if deltas[1].(float64) != 200 {
		t.Errorf("day2 delta: want 200, got %v", deltas[1])
	}
}
