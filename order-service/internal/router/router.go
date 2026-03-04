package router

import (
	"order-service/internal/handlers"

	"github.com/gin-gonic/gin"
)

func InitRouter(orderHandler *handlers.Handler) *gin.Engine {
	router := gin.Default()

	order := router.Group("/order")
	{
		order.GET("/:order_uid", orderHandler.GetOrderByUID)
	}

	router.Static("/static", "./web/static")
	router.StaticFile("/", "./web/static/index.html")

	return router
}
