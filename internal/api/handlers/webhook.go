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
	"io"
	"net/http"

	"github.com/urustack/uruflow/internal/services"
	"github.com/urustack/uruflow/pkg/helper"
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
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	body, err := io.ReadAll(r.Body)
	if err != nil {
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

	helper.WriteError(w, http.StatusBadRequest, "unsupported webhook source")
}

func (h *WebhookHandler) handleGitHub(w http.ResponseWriter, r *http.Request, body []byte) {
	signature := r.Header.Get("X-Hub-Signature-256")
	if !h.webhookService.ValidateGitHubSignature(body, signature) {
		helper.WriteError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "push" {
		helper.WriteJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	_, err := h.webhookService.ProcessGitHubPush(body)
	if err != nil {
		helper.WriteJSON(w, http.StatusOK, map[string]string{"status": "skipped"})
		return
	}

	helper.WriteJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *WebhookHandler) handleGitLab(w http.ResponseWriter, r *http.Request, body []byte) {
	token := r.Header.Get("X-Gitlab-Token")
	if !h.webhookService.ValidateGitLabToken(token) {
		helper.WriteError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	event := r.Header.Get("X-Gitlab-Event")
	if event != "Push Hook" {
		helper.WriteJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	_, err := h.webhookService.ProcessGitLabPush(body)
	if err != nil {
		helper.WriteJSON(w, http.StatusOK, map[string]string{"status": "skipped"})
		return
	}

	helper.WriteJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func isGitHub(r *http.Request) bool {
	return r.Header.Get("X-GitHub-Event") != ""
}

func isGitLab(r *http.Request) bool {
	return r.Header.Get("X-Gitlab-Event") != ""
}
