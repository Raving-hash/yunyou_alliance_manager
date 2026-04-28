package repository_test

import (
	"testing"
	"time"
	"yunyoumanager/internal/model"
	"yunyoumanager/internal/repository"
	"yunyoumanager/internal/testutil"
)

func newRepo(t *testing.T) *repository.Repository {
	t.Helper()
	return repository.New(testutil.NewDB(t))
}

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

// ── UpsertMember ────────────────────────────────────────────────────────────

func TestUpsertMember_CreatesNew(t *testing.T) {
	repo := newRepo(t)
	id, err := repo.UpsertMember("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestUpsertMember_Idempotent(t *testing.T) {
	repo := newRepo(t)
	id1, _ := repo.UpsertMember("Alice")
	id2, _ := repo.UpsertMember("Alice")
	if id1 != id2 {
		t.Fatalf("expected same ID, got %d and %d", id1, id2)
	}
}

// ── UpsertDailyRecord ────────────────────────────────────────────────────────

func TestUpsertDailyRecord_Insert(t *testing.T) {
	repo := newRepo(t)
	id, _ := repo.UpsertMember("Alice")
	err := repo.UpsertDailyRecord(model.DailyRecord{
		MemberID: id, Date: day("2024-01-01"),
		MilitaryMerit: 1000, Prosperity: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpsertDailyRecord_OverwritesSameDate(t *testing.T) {
	repo := newRepo(t)
	id, _ := repo.UpsertMember("Alice")
	rec := model.DailyRecord{MemberID: id, Date: day("2024-01-01"), MilitaryMerit: 1000, Prosperity: 500}
	repo.UpsertDailyRecord(rec)

	rec.MilitaryMerit = 2000
	if err := repo.UpsertDailyRecord(rec); err != nil {
		t.Fatal(err)
	}

	records, _ := repo.GetChartData(id)
	if records[0].MilitaryMerit != 2000 {
		t.Fatalf("expected 2000, got %d", records[0].MilitaryMerit)
	}
}

// ── ListMembers ──────────────────────────────────────────────────────────────

func TestListMembers_OrderedByUsername(t *testing.T) {
	repo := newRepo(t)
	for _, name := range []string{"Charlie", "Alice", "Bob"} {
		repo.UpsertMember(name)
	}
	members, err := repo.ListMembers()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Alice", "Bob", "Charlie"}
	for i, m := range members {
		if m.Username != want[i] {
			t.Errorf("pos %d: want %s, got %s", i, want[i], m.Username)
		}
	}
}

// ── GetChartData ─────────────────────────────────────────────────────────────

func TestGetChartData_OrderedByDate(t *testing.T) {
	repo := newRepo(t)
	id, _ := repo.UpsertMember("Alice")
	for _, d := range []string{"2024-01-03", "2024-01-01", "2024-01-02"} {
		repo.UpsertDailyRecord(model.DailyRecord{MemberID: id, Date: day(d), MilitaryMerit: 1, Prosperity: 1})
	}
	records, err := repo.GetChartData(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if records[0].Date.After(records[1].Date) || records[1].Date.After(records[2].Date) {
		t.Error("records not in ascending date order")
	}
}

// ── GetMemberByUsername ───────────────────────────────────────────────────────

func TestGetMemberByUsername_Found(t *testing.T) {
	repo := newRepo(t)
	repo.UpsertMember("Alice")
	m, err := repo.GetMemberByUsername("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if m.Username != "Alice" {
		t.Errorf("want Alice, got %s", m.Username)
	}
}

func TestGetMemberByUsername_NotFound(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.GetMemberByUsername("Ghost")
	if err == nil {
		t.Fatal("expected error for missing member")
	}
}

// ── GetMembersOverview ────────────────────────────────────────────────────────

func seedMember(t *testing.T, repo *repository.Repository, name string, days []struct {
	date          string
	military, pro int64
}) {
	t.Helper()
	id, _ := repo.UpsertMember(name)
	for _, d := range days {
		repo.UpsertDailyRecord(model.DailyRecord{
			MemberID: id, Date: day(d.date),
			MilitaryMerit: d.military, Prosperity: d.pro,
		})
	}
}

func TestGetMembersOverview_CurrentValues(t *testing.T) {
	repo := newRepo(t)
	seedMember(t, repo, "Alice", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 1000, 200},
		{"2024-01-02", 1500, 300},
	})
	ovs, err := repo.GetMembersOverview()
	if err != nil {
		t.Fatal(err)
	}
	if len(ovs) != 1 {
		t.Fatalf("expected 1, got %d", len(ovs))
	}
	ov := ovs[0]
	if ov.CurrentMilitary != 1500 {
		t.Errorf("CurrentMilitary: want 1500, got %d", ov.CurrentMilitary)
	}
	if ov.CurrentProsperity != 300 {
		t.Errorf("CurrentProsperity: want 300, got %d", ov.CurrentProsperity)
	}
}

func TestGetMembersOverview_GrowthRateNilWithLessThan3Days(t *testing.T) {
	repo := newRepo(t)
	seedMember(t, repo, "Alice", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 1000, 200},
		{"2024-01-02", 1200, 300},
	})
	ovs, _ := repo.GetMembersOverview()
	if ovs[0].GrowthRate != nil {
		t.Error("expected nil GrowthRate with only 2 days of data")
	}
}

func TestGetMembersOverview_GrowthRateCalculated(t *testing.T) {
	repo := newRepo(t)
	// delta0 = 1500-1200=300, delta1 = 1200-1000=200
	// rate = (300-200)/|200| * 100 = 50%
	seedMember(t, repo, "Alice", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 1000, 200},
		{"2024-01-02", 1200, 300},
		{"2024-01-03", 1500, 400},
	})
	ovs, _ := repo.GetMembersOverview()
	if ovs[0].GrowthRate == nil {
		t.Fatal("expected non-nil GrowthRate")
	}
	got := *ovs[0].GrowthRate
	if got < 49.9 || got > 50.1 {
		t.Errorf("GrowthRate: want ~50, got %.4f", got)
	}
}

func TestGetMembersOverview_RecentAtMost3(t *testing.T) {
	repo := newRepo(t)
	seedMember(t, repo, "Alice", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 100, 10},
		{"2024-01-02", 200, 20},
		{"2024-01-03", 300, 30},
		{"2024-01-04", 400, 40},
		{"2024-01-05", 500, 50},
	})
	ovs, _ := repo.GetMembersOverview()
	if len(ovs[0].Recent) != 3 {
		t.Errorf("expected 3 recent records, got %d", len(ovs[0].Recent))
	}
}

func TestGetMembersOverview_MilitaryDelta(t *testing.T) {
	repo := newRepo(t)
	seedMember(t, repo, "Alice", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 1000, 200},
		{"2024-01-02", 1300, 300},
	})
	ovs, _ := repo.GetMembersOverview()
	rec := ovs[0].Recent
	// rec[0] = newest (2024-01-02), rec[1] = older (2024-01-01)
	if rec[0].MilitaryDelta == nil {
		t.Fatal("expected non-nil delta for newest record")
	}
	if *rec[0].MilitaryDelta != 300 {
		t.Errorf("MilitaryDelta: want 300, got %d", *rec[0].MilitaryDelta)
	}
	if rec[1].MilitaryDelta != nil {
		t.Error("oldest record should have nil delta")
	}
}

// ── GetAllianceDailyTotals ───────────────────────────────────────────────────

func TestGetAllianceDailyTotals(t *testing.T) {
	repo := newRepo(t)
	seedMember(t, repo, "Alice", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 1000, 100},
		{"2024-01-02", 2000, 200},
	})
	seedMember(t, repo, "Bob", []struct{ date string; military, pro int64 }{
		{"2024-01-01", 500, 50},
		{"2024-01-02", 600, 60},
	})
	totals, err := repo.GetAllianceDailyTotals()
	if err != nil {
		t.Fatal(err)
	}
	if len(totals) != 2 {
		t.Fatalf("expected 2 totals, got %d", len(totals))
	}
	if totals[0].TotalMilitary != 1500 {
		t.Errorf("day1 TotalMilitary: want 1500, got %d", totals[0].TotalMilitary)
	}
	if totals[1].TotalMilitary != 2600 {
		t.Errorf("day2 TotalMilitary: want 2600, got %d", totals[1].TotalMilitary)
	}
}
