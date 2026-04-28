package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"yunyoumanager/internal/model"
	"yunyoumanager/internal/repository"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	repo *repository.Repository
}

func NewUploadHandler(repo *repository.Repository) *UploadHandler {
	return &UploadHandler{repo: repo}
}

// Upload handles POST /api/upload
// Expects multipart form with field "file" (csv) and optional "date" (YYYY-MM-DD).
// CSV columns (comma-separated): 用户名,武勋,繁荣
func (h *UploadHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	// Parse optional date param, default to today.
	uploadDate := repository.TruncateDate(time.Now())
	if dateStr := c.PostForm("date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			uploadDate = t
		}
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件"})
		return
	}
	defer src.Close()

	rows, err := readCSV(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析CSV文件: " + err.Error()})
		return
	}

	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "数据行不足，请检查表头"})
		return
	}

	colIdx, err := parseHeader(rows[0])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var imported, skipped int
	for i, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		username := ""
		if colIdx.username < len(row) {
			username = strings.TrimSpace(row[colIdx.username])
		}
		if username == "" {
			skipped++
			continue
		}

		var militaryMerit, prosperity int64
		if colIdx.military < len(row) {
			militaryMerit, _ = strconv.ParseInt(strings.TrimSpace(row[colIdx.military]), 10, 64)
		}
		if colIdx.prosperity < len(row) {
			prosperity, _ = strconv.ParseInt(strings.TrimSpace(row[colIdx.prosperity]), 10, 64)
		}

		memberID, err := h.repo.UpsertMember(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("第%d行处理失败: %v", i+2, err)})
			return
		}

		rec := model.DailyRecord{
			MemberID:      memberID,
			Date:          uploadDate,
			MilitaryMerit: militaryMerit,
			Prosperity:    prosperity,
		}
		if err := h.repo.UpsertDailyRecord(rec); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("第%d行写入失败: %v", i+2, err)})
			return
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "上传成功",
		"date":     uploadDate.Format("2006-01-02"),
		"imported": imported,
		"skipped":  skipped,
	})
}

func readCSV(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	return reader.ReadAll()
}

type colIndex struct {
	username   int
	military   int
	prosperity int
}

func parseHeader(header []string) (colIndex, error) {
	idx := colIndex{username: -1, military: -1, prosperity: -1}
	for i, cell := range header {
		switch strings.TrimSpace(cell) {
		case "用户名":
			idx.username = i
		case "武勋":
			idx.military = i
		case "繁荣":
			idx.prosperity = i
		}
	}
	if idx.username == -1 || idx.military == -1 || idx.prosperity == -1 {
		return idx, fmt.Errorf("表头必须包含：用户名、武勋、繁荣")
	}
	return idx, nil
}
