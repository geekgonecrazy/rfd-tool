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

	// OIDC endpoints
	router.GET("/oidc/login", controllers.OIDCAuthorizationURLHandler)
	router.GET("/oidc/callback", controllers.OIDCCallbackHandler)

	router.Use(getSessionFromCookieOrHeader)
	router.Use(_debugBypassEmptyLogin)

	api := router.Group("/api/v1")
	{
		// Create/Update RFD by its ID happens from rfd-client currently
		api.POST("/rfds/:id", requireAPISecret, controllers.CreateOrUpdateRFDHandler)

		api.Use(requireSession)
		api.GET("/rfds", controllers.GetRFDsHandler)
		api.POST("/rfds", controllers.CreateRFDHandler)
		api.GET("/rfds/:id", controllers.GetRFDHandler)

		api.GET("/tags", controllers.GetTagsHandler)
		api.GET("/tags/:tag/rfds", controllers.GetRFDsForTagHandler)
	}

	// Server Side Rendered Pages
	router.GET("/tag/:tag", requireSession, controllers.TagListPageHandler)
	router.GET("/:id", requireSession, controllers.RFDPageHandler)

	router.GET("/create", requireSession, controllers.RFDCreatePageHandler)
	router.GET("/created", requireSession, controllers.RFDCreatedPageHandler)
	router.POST("/created", requireSession, controllers.RFDCreatedPageHandler)

	router.GET("/login", controllers.LoginPageHandler)

	router.GET("/", controllers.DefaultRouteHandler)
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
