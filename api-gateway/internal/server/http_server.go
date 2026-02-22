package server

import (
	"building-services/api-gateway/config"
	"log"

	"github.com/gin-gonic/gin"
)

type HTTPServer struct {
	cfg            config.ServerConfig
	registerRoutes func(r *gin.Engine)
}

func NewServer(cfg config.ServerConfig, registerRoutes func(*gin.Engine)) *HTTPServer {
	return &HTTPServer{
		cfg:            cfg,
		registerRoutes: registerRoutes,
	}
}

func (s *HTTPServer) StartServer() error {
	router := gin.Default()
	s.registerRoutes(router)

	log.Printf("Gateway running on :%s", s.cfg.Port)
	return router.Run(":" + s.cfg.Port)
}
