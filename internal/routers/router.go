package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/loggers"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

var logger = loggers.GetLogger()

func InitRouter() *gin.Engine {

	logger = loggers.GetLogger()

	if !configs.GetAppConfig().Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	//@todo: init common user identity authentication logic here
	r := gin.New()

	r.MaxMultipartMemory = 8 << 20 // 8 MiB

	r.GET("/swagger/*any", ginSwagger.DisablingWrapHandler(swaggerFiles.Handler, "DISABLE_SWAGGER"))

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	r.Use(cors.Default())
	//r.Use(Auth())

	//r.NoMethod(HandleNotFound)
	//r.NoRoute(HandleNotFound)

	r.Use(loggers.GinLogger(logger))
	r.Use(gin.Recovery())



	return r
}
