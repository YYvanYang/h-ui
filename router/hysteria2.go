package router

import (
	"github.com/gin-gonic/gin"
	"h-ui/controller"
)

func initHysteria2AuthRouter(hysteria2Api *gin.RouterGroup) {
	hysteria2 := hysteria2Api.Group("/hysteria2")
	{
		hysteria2.POST("/auth", controller.Hysteria2Auth)
	}
}

func initHysteria2Router(hysteria2Api *gin.RouterGroup) {
	hysteria2 := hysteria2Api.Group("/hysteria2")
	{
		hysteria2.POST("/hysteria2Kick", controller.Hysteria2Kick)
		hysteria2.POST("/changeHysteria2Version", controller.ChangeHysteria2Version)
		hysteria2.GET("/listRelease", controller.ListRelease)
		hysteria2.GET("/hysteria2Url", controller.Hysteria2Url)
	}
}
