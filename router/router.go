package router

import (
	"github.com/geekgonecrazy/rfd-tool/controllers"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

func Run() error {
	runHealthMetricsRouter()

	router := gin.Default()

	router.LoadHTMLGlob("templates/*")
	router.Use(static.Serve("/assets", static.LocalFile("./assets", false)))
	router.GET("/assets/logo.svg", controllers.ServeLogoSVGHandler)

	//router.Use(getSessionFromCookie)

	// Construct oauth authorize url and redirect
	router.GET("/oidc/login", controllers.OIDCAuthorizationURLHandler)

	// oidc oauth callback url
	router.GET("/oidc/callback", controllers.OIDCCallbackHandler)

	router.Use(getSessionFromCookie)

	// Create RFD
	//router.POST("/v1/rfds", CreateRFDHandler)
	router.POST("/api/v1/rfds/:id", requireAPISecret, controllers.CreateRFDHandler)
	router.GET("/api/v1/rfds/:id", requireAPISecret, controllers.GetRFDHandler)

	router.GET("/tag/:tag", requireSession, controllers.TagListPageHandler)

	router.GET("/", controllers.DefaultRouteHandler)
	router.GET("/:id", requireSession, controllers.GetRFDPageHandler)

	router.GET("/login", controllers.LoginPageHandler)

	router.NoRoute(controllers.DefaultRouteHandler)

	if err := router.Run(":8877"); err != nil {
		return err
	}

	return nil
}

func runHealthMetricsRouter() {
	healthMetricsRouter := gin.New()
	healthMetricsRouter.Use(gin.Recovery())

	healthMetricsRouter.GET("/health", controllers.LivenessCheckHandler)

	go healthMetricsRouter.Run(":8080")
}
