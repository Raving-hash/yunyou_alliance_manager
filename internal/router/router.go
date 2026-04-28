package router

import (
	"yunyoumanager/internal/handler"
	"yunyoumanager/internal/repository"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Setup(repo *repository.Repository) *gin.Engine {
	r := gin.Default()

	// Allow all origins during development.
	r.Use(cors.Default())

	uploadH := handler.NewUploadHandler(repo)
	memberH := handler.NewMemberHandler(repo)
	chartH := handler.NewChartHandler(repo)

	api := r.Group("/api")
	{
		api.POST("/upload", uploadH.Upload)
		api.GET("/members", memberH.List)
		api.GET("/members/overview", memberH.Overview)
		api.GET("/chart/alliance", chartH.GetAllianceTotals)
		api.GET("/chart/:username", chartH.GetByMember)
	}

	return r
}
