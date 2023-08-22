package routes

import (
	"github.com/gergpol1998/gin-mongo-api/controllers"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupRouter(collection *mongo.Collection) *gin.Engine {
	r := gin.Default()

	userController := controllers.NewUserController(collection)

	userRoutes := r.Group("/")
	{
		userRoutes.POST("/user", userController.Create)
		userRoutes.PUT("/user/:user_id", userController.Update)
		userRoutes.GET("/users", userController.List)
		userRoutes.GET("/user/:user_id", userController.GetByID)
		userRoutes.DELETE("/user/:user_id", userController.DeleteByID)
	}

	return r
}
