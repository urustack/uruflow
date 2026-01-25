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

package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/urustack/uruflow/internal/services"
	"github.com/urustack/uruflow/pkg/helper"
	"github.com/urustack/uruflow/pkg/logger"
)

type WebhookHandler struct {
	webhookService *services.WebhookService
}

func NewWebhookHandler(webhookService *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	logger.Info("[WEBHOOK] Received webhook from %s", r.RemoteAddr)

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("[WEBHOOK] Failed to read body: %v", err)
		helper.WriteError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	if isGitHub(r) {
		h.handleGitHub(w, r, body)
		return
	}

	if isGitLab(r) {
		h.handleGitLab(w, r, body)
		return
	}

	logger.Warn("[WEBHOOK] Unsupported webhook source from %s", r.RemoteAddr)
	helper.WriteError(w, http.StatusBadRequest, "unsupported webhook source")
}

func (h *WebhookHandler) handleGitHub(w http.ResponseWriter, r *http.Request, body []byte) {
	signature := r.Header.Get("X-Hub-Signature-256")

	logger.Debug("[WEBHOOK] GitHub webhook received, validating signature")

	if !h.webhookService.ValidateGitHubSignature(body, signature) {
		logger.Warn("[WEBHOOK] GitHub signature validation failed from %s", r.RemoteAddr)
		helper.WriteError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "push" {
		logger.Debug("[WEBHOOK] GitHub event '%s' ignored (not a push event)", event)
		helper.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ignored",
			"reason": fmt.Sprintf("event type '%s' not supported", event),
		})
		return
	}

	result, err := h.webhookService.ProcessGitHubPush(body)
	if err != nil {
		logger.Error("[WEBHOOK] GitHub deployment failed: %v", err)

		helper.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	logger.Info("[WEBHOOK] GitHub deployment triggered: repo=%s branch=%s commit=%s deployment_id=%s",
		result.Repository, result.Branch, result.Commit, result.Deployment.ID)

	helper.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "accepted",
		"deployment_id": result.Deployment.ID,
		"repository":    result.Repository,
		"branch":        result.Branch,
		"commit":        result.Commit,
	})
}

func (h *WebhookHandler) handleGitLab(w http.ResponseWriter, r *http.Request, body []byte) {
	token := r.Header.Get("X-Gitlab-Token")

	logger.Debug("[WEBHOOK] GitLab webhook received, validating token")

	if !h.webhookService.ValidateGitLabToken(token) {
		logger.Warn("[WEBHOOK] GitLab token validation failed from %s", r.RemoteAddr)
		helper.WriteError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	event := r.Header.Get("X-Gitlab-Event")
	if event != "Push Hook" {
		logger.Debug("[WEBHOOK] GitLab event '%s' ignored (not a push event)", event)
		helper.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ignored",
			"reason": fmt.Sprintf("event type '%s' not supported", event),
		})
		return
	}

	result, err := h.webhookService.ProcessGitLabPush(body)
	if err != nil {
		logger.Error("[WEBHOOK] GitLab deployment failed: %v", err)

		helper.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	logger.Info("[WEBHOOK] GitLab deployment triggered: repo=%s branch=%s commit=%s deployment_id=%s",
		result.Repository, result.Branch, result.Commit, result.Deployment.ID)

	helper.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "accepted",
		"deployment_id": result.Deployment.ID,
		"repository":    result.Repository,
		"branch":        result.Branch,
		"commit":        result.Commit,
	})
}

func isGitHub(r *http.Request) bool {
	return r.Header.Get("X-GitHub-Event") != ""
}

func isGitLab(r *http.Request) bool {
	return r.Header.Get("X-Gitlab-Event") != ""
}
