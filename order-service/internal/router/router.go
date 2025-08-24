package router

import (
	"github.com/gin-gonic/gin"
	"order-service/internal/handlers"
)

func InitRouter(orderHandler *handlers.Handler) *gin.Engine {
	router := gin.Default()

	order := router.Group("/order")
	{
		order.GET("/:order_uid", orderHandler.GetOrderByUID)
	}

	return router
}
