package main

import (
	"building-services/api-gateway/config"
	"building-services/api-gateway/internal/app"
	"log"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")

	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	appGateway, err := app.NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to configure app: %v", err)
	}

	if err := appGateway.Run(); err != nil {
		log.Fatalf("Failed to run app: %v", err)
	}

}
