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

package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/models"
)

type WebhookService struct {
	cfg           *config.Config
	deployService *DeploymentService
}

func NewWebhookService(cfg *config.Config, ds *DeploymentService) *WebhookService {
	return &WebhookService{
		cfg:           cfg,
		deployService: ds,
	}
}

type WebhookResult struct {
	Repository string
	Branch     string
	Commit     string
	Deployment *models.Deployment
}

type GitHubPushPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	HeadCommit struct {
		ID string `json:"id"`
	} `json:"head_commit"`
}

type GitLabPushPayload struct {
	Ref     string `json:"ref"`
	Project struct {
		Name string `json:"name"`
	} `json:"project"`
	Commits []struct {
		ID string `json:"id"`
	} `json:"commits"`
}

func (s *WebhookService) ValidateGitHubSignature(payload []byte, signature string) bool {
	if s.cfg.Webhook.Secret == "" {
		return true
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sig := strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(s.cfg.Webhook.Secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func (s *WebhookService) ValidateGitLabToken(token string) bool {
	if s.cfg.Webhook.Secret == "" {
		return true
	}
	return token == s.cfg.Webhook.Secret
}

func (s *WebhookService) ProcessGitHubPush(payload []byte) (*WebhookResult, error) {
	var data GitHubPushPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("invalid payload")
	}

	branch := extractBranch(data.Ref)
	if branch == "" {
		return nil, fmt.Errorf("invalid ref")
	}

	repoName := data.Repository.Name
	repo := s.cfg.GetRepository(repoName)
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	if repo.Branch != branch {
		return nil, ErrBranchNotConfigured
	}

	if !repo.AutoDeploy {
		return nil, ErrAutoDeployDisabled
	}

	deploy, err := s.deployService.TriggerDeploy(repo.AgentID, repoName, branch, data.HeadCommit.ID, "webhook")
	if err != nil {
		return nil, err
	}

	return &WebhookResult{
		Repository: repoName,
		Branch:     branch,
		Commit:     data.HeadCommit.ID[:7],
		Deployment: deploy,
	}, nil
}

func (s *WebhookService) ProcessGitLabPush(payload []byte) (*WebhookResult, error) {
	var data GitLabPushPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("invalid payload")
	}

	branch := extractBranch(data.Ref)
	if branch == "" {
		return nil, fmt.Errorf("invalid ref")
	}

	repoName := data.Project.Name
	repo := s.cfg.GetRepository(repoName)
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	if repo.Branch != branch {
		return nil, ErrBranchNotConfigured
	}

	if !repo.AutoDeploy {
		return nil, ErrAutoDeployDisabled
	}

	commitID := ""
	if len(data.Commits) > 0 {
		commitID = data.Commits[0].ID
	}

	deploy, err := s.deployService.TriggerDeploy(repo.AgentID, repoName, branch, commitID, "webhook")
	if err != nil {
		return nil, err
	}

	return &WebhookResult{
		Repository: repoName,
		Branch:     branch,
		Commit:     commitID,
		Deployment: deploy,
	}, nil
}

func extractBranch(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ""
}
