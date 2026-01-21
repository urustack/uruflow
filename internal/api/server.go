/*
 * Copyright (C) 2026 Mustafa Naseer (Mustafa Gaeed)
 *
 * This file is part of uruflow.
 *
 * uruflow is free software: you can redistribute it and/or modify
 * it under the terms of the MIT License as described in the
 * LICENSE file distributed with this project.
 *
 * uruflow is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * MIT License for more details.
 *
 * You should have received a copy of the MIT License
 * along with uruflow. If not, see the LICENSE file in the project root.
 */

package api

import (
	"context"
	"fmt"
	"github.com/urustack/uruflow/internal/api/middleware"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/urustack/uruflow/internal/api/handlers"
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/services"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tcp"
)

type Server struct {
	cfg            *config.Config
	store          storage.Store
	httpServer     *http.Server
	tcpServer      *tcp.Server
	deployService  *services.DeploymentService
	webhookService *services.WebhookService
}

func NewServer(cfg *config.Config, store storage.Store) *Server {
	tcpServer := tcp.NewServer(cfg, store)
	deployService := services.NewDeploymentService(cfg, store, tcpServer)
	webhookService := services.NewWebhookService(cfg, deployService)

	return &Server{
		cfg:            cfg,
		store:          store,
		tcpServer:      tcpServer,
		deployService:  deployService,
		webhookService: webhookService,
	}
}

func (s *Server) Start() error {
	if err := s.tcpServer.Start(); err != nil {
		return fmt.Errorf("tcp server: %w", err)
	}

	router := s.setupRoutes()
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.HTTPPort),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[HTTP] Webhook listener on %s:%d", s.cfg.Server.Host, s.cfg.Server.HTTPPort)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTP] Error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.tcpServer.Stop()

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) setupRoutes() http.Handler {
	r := mux.NewRouter()
	webhookHandler := handlers.NewWebhookHandler(s.webhookService)
	r.HandleFunc(s.cfg.Webhook.Path, webhookHandler.Handle).Methods("POST")
	return middleware.Recovery(middleware.Logging(r))
}

func (s *Server) GetStore() storage.Store {
	return s.store
}

func (s *Server) GetTCPServer() *tcp.Server {
	return s.tcpServer
}

func (s *Server) GetDeployService() *services.DeploymentService {
	return s.deployService
}
