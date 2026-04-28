# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

云游联盟管理系统 — An alliance management web app. Admins upload Excel files with member data (username, 武勋, 繁荣), and the system generates daily trend charts.

## Commands

### Backend (Go)
```bash
go run main.go          # Start server on :8080
go build ./...          # Compile check
go mod tidy             # Sync dependencies
```

### Frontend (Vue 3 + Vite, in /frontend)
```bash
npm run dev             # Dev server on :5173 (proxies /api → :8080)
npm run build           # Production build → dist/
```

## Architecture

```
yunyoumanager/
├── main.go                         # DB init + server bootstrap
├── config/config.go                # Port (:8080), DBPath (data/yunyou.db)
├── internal/
│   ├── model/model.go              # Member, DailyRecord (GORM structs)
│   ├── repository/repository.go    # All DB operations
│   ├── handler/
│   │   ├── upload.go               # POST /api/upload — Excel parse & upsert
│   │   ├── member.go               # GET /api/members
│   │   └── chart.go                # GET /api/chart/:id, /api/chart/all
│   └── router/router.go            # Gin route wiring + CORS
├── data/yunyou.db                  # SQLite (auto-created on first run)
└── frontend/
    ├── src/
    │   ├── api.js                  # Axios wrappers for all endpoints
    │   ├── App.vue                 # Root layout
    │   └── components/
    │       ├── UploadPanel.vue     # Excel upload form
    │       └── MemberChart.vue     # ECharts trend charts
    └── vite.config.js              # Proxies /api to :8080 in dev
```

## Key Design Decisions

- **SQLite** single-file DB, stored in `data/` (no external DB needed)
- **DailyRecord** has a unique constraint on `(member_id, date)` — re-uploading same date overwrites
- **Excel format**: first row must be headers containing `用户名`, `武勋`, `繁荣` (column order flexible)
- Upload accepts optional `date` form field (YYYY-MM-DD); defaults to server today
- Frontend uses Vite proxy in dev — `api.js` uses relative `/api` paths in dev, full URL only needed in prod
