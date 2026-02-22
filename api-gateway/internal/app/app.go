package app

import (
	"building-services/api-gateway/config"
	"building-services/api-gateway/internal/clients"
	"building-services/api-gateway/internal/handler"
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
	authHandler := handler.NewAuthHandler(authClient)

	authHealthClient := healthpb.NewHealthClient(authConn)

	healthHandler := handler.NewHealthHandler([]handler.ServiceHealth{
		{
			Name:   "auth",
			Client: authHealthClient,
		},
	})

	registerRoutes := func(r *gin.Engine) {
		authHandler.RegisterRoutes(r)
		healthHandler.RegisterRoutes(r)
	}

	srv := server.NewServer(cfg.Server, registerRoutes)

	return &App{server: srv}, nil
}

func (a *App) Run() error {

	return a.server.StartServer()

}
