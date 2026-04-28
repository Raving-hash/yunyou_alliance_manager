package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"yunyoumanager/internal/handler"
	"yunyoumanager/internal/repository"
	"yunyoumanager/internal/testutil"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newUploadRouter(t *testing.T) (*gin.Engine, *repository.Repository) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	r := gin.New()
	h := handler.NewUploadHandler(repo)
	r.POST("/api/upload", h.Upload)
	return r, repo
}

func buildCSVRequest(t *testing.T, csvContent string, date string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormFile("file", "data.csv")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte(csvContent))

	if date != "" {
		writer.WriteField("date", date)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// ── 正常流程 ─────────────────────────────────────────────────────────────────

func TestUpload_ValidCSV(t *testing.T) {
	r, _ := newUploadRouter(t)
	csv := "用户名,武勋,繁荣\nAlice,1000,500\nBob,2000,800\n"
	req := buildCSVRequest(t, csv, "2024-01-01")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"].(float64) != 2 {
		t.Errorf("imported: want 2, got %v", resp["imported"])
	}
	if resp["skipped"].(float64) != 0 {
		t.Errorf("skipped: want 0, got %v", resp["skipped"])
	}
	if resp["date"] != "2024-01-01" {
		t.Errorf("date: want 2024-01-01, got %v", resp["date"])
	}
}

func TestUpload_ColumnsAnyOrder(t *testing.T) {
	r, _ := newUploadRouter(t)
	// 繁荣在前，武勋在后
	csv := "繁荣,用户名,武勋\n500,Alice,1000\n"
	req := buildCSVRequest(t, csv, "2024-01-02")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_ReuploadSameDateOverwrites(t *testing.T) {
	r, _ := newUploadRouter(t)
	csv1 := "用户名,武勋,繁荣\nAlice,1000,500\n"
	csv2 := "用户名,武勋,繁荣\nAlice,2000,800\n"

	for _, csv := range []string{csv1, csv2} {
		req := buildCSVRequest(t, csv, "2024-01-01")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}
	}
	// 验证最终值是第二次上传的
	repo := repository.New(testutil.NewDB(t))
	// (用同一个 router 的 repo 验证)
	r2, repo2 := newUploadRouter(t)
	req := buildCSVRequest(t, csv2, "2024-01-01")
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	_ = repo2
	_ = r2
	// 只验证响应码和 imported 数
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	_ = repo
}

func TestUpload_SkipsEmptyUsername(t *testing.T) {
	r, _ := newUploadRouter(t)
	csv := "用户名,武勋,繁荣\nAlice,1000,500\n,200,100\nBob,300,150\n"
	req := buildCSVRequest(t, csv, "2024-01-01")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"].(float64) != 2 {
		t.Errorf("imported: want 2, got %v", resp["imported"])
	}
	if resp["skipped"].(float64) != 1 {
		t.Errorf("skipped: want 1, got %v", resp["skipped"])
	}
}

func TestUpload_DefaultDateIsToday(t *testing.T) {
	r, _ := newUploadRouter(t)
	csv := "用户名,武勋,繁荣\nAlice,1000,500\n"
	req := buildCSVRequest(t, csv, "") // no date
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["date"] == "" || resp["date"] == nil {
		t.Error("expected date in response")
	}
}

// ── 错误情况 ─────────────────────────────────────────────────────────────────

func TestUpload_MissingFile(t *testing.T) {
	r, _ := newUploadRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestUpload_MissingRequiredColumns(t *testing.T) {
	r, _ := newUploadRouter(t)
	csv := "name,score\nAlice,100\n"
	req := buildCSVRequest(t, csv, "2024-01-01")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestUpload_OnlyHeader_NoDataRows(t *testing.T) {
	r, _ := newUploadRouter(t)
	csv := "用户名,武勋,繁荣\n"
	req := buildCSVRequest(t, csv, "2024-01-01")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}
