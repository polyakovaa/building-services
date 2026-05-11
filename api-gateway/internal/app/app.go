package app

import (
	"building-services/api-gateway/config"
	"building-services/api-gateway/internal/clients"
	"building-services/api-gateway/internal/handler"
	"building-services/api-gateway/internal/middleware"
	"building-services/api-gateway/internal/server"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/gin-gonic/gin"
)

type App struct {
	server *server.HTTPServer
}

func NewApp(cfg *config.GatewayConfig) (*App, error) {
	authServiceCfg, err := cfg.GetServiceConfig("auth")
	if err != nil {
		return nil, err
	}

	authClient, authConn, err := clients.NewAuthClient(authServiceCfg.Address)
	if err != nil {
		return nil, err
	}
	authHandler := handler.NewAuthHandler(authClient, cfg.JWTSecret)
	authHealthClient := healthpb.NewHealthClient(authConn)

	projectServiceCfg, err := cfg.GetServiceConfig("project")
	if err != nil {
		return nil, err
	}

	projectClient, projectConn, err := clients.NewProjectClient(projectServiceCfg.Address)
	if err != nil {
		return nil, err
	}
	projectHandler := handler.NewProjectHandler(projectClient)
	projectHealthClient := healthpb.NewHealthClient(projectConn)

	analyticsServiceCfg, err := cfg.GetServiceConfig("analytics")
	if err != nil {
		return nil, err
	}

	analyticsClient, analyticsConn, err := clients.NewAnalyticsClient(analyticsServiceCfg.Address)
	if err != nil {
		return nil, err
	}
	analyticsHandler := handler.NewAnalyticsHandler(analyticsClient)
	analyticsHealthClient := healthpb.NewHealthClient(analyticsConn)

	healthHandler := handler.NewHealthHandler([]handler.ServiceHealth{
		{Name: "auth", Client: authHealthClient},
		{Name: "project", Client: projectHealthClient},
		{Name: "analytics", Client: analyticsHealthClient},
	})

	authMiddleware := middleware.AuthRequired(cfg.JWTSecret)
	adminHandler := handler.NewAdminHandler(authClient)

	registerRoutes := func(r *gin.Engine) {
		authHandler.RegisterRoutes(r)
		healthHandler.RegisterRoutes(r)
		protected := r.Group("/api")
		protected.Use(authMiddleware)
		{
			projectHandler.RegisterRoutes(protected)
			protected.GET("/users/me", authHandler.GetInfo)
			protected.GET("/users/:id", projectHandler.GetUserByID)
			protected.GET("/users/by-email", projectHandler.GetUserByEmail)
			analyticsHandler.RegisterRoutes(protected)

		}
		adminHandler.RegisterRoutes(r)
		serveFrontend(r)
	}

	srv := server.NewServer(cfg.Server, registerRoutes)

	return &App{server: srv}, nil
}

func (a *App) Run() error {

	return a.server.StartServer()

}

func serveFrontend(r *gin.Engine) {
	r.Static("/static", "./frontend/static")
	r.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})
	r.GET("/dashboard", func(c *gin.Context) {
		c.File("./frontend/dashboard.html")
	})
	r.GET("/projects", func(c *gin.Context) {
		c.File("./frontend/projects.html")
	})

	r.GET("/project/:id", func(c *gin.Context) {
		c.File("./frontend/project.html")
	})
	r.GET("/tasks", func(c *gin.Context) {
		c.File("./frontend/tasks.html")
	})
	r.GET("/profile", func(c *gin.Context) {
		c.File("./frontend/profile.html")
	})

	r.GET("/task/:id", func(c *gin.Context) {
		c.File("./frontend/task.html")
	})

}
