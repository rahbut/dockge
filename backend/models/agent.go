package models

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rahbut/dockge/backend/db"
)

// AgentModel wraps a db.Agent with helpers.
type AgentModel struct {
	db.Agent
}

// Endpoint returns the host portion of the agent URL (used as the unique key
// by the frontend and agent manager, matching the TS `endpoint` getter).
func (a *AgentModel) Endpoint() string {
	u, err := url.Parse(a.URL)
	if err != nil {
		return ""
	}
	return u.Host
}

// ToJSON returns a map suitable for JSON serialisation to the frontend.
func (a *AgentModel) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"id":       a.ID,
		"url":      a.URL,
		"username": a.Username,
		"endpoint": a.Endpoint(),
	}
}

// GetAgentList returns all agents keyed by endpoint string.
func GetAgentList(ctx context.Context) (map[string]*AgentModel, error) {
	rows, err := db.GetAllAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("get agents: %w", err)
	}
	out := make(map[string]*AgentModel, len(rows))
	for i := range rows {
		a := &AgentModel{Agent: rows[i]}
		out[a.Endpoint()] = a
	}
	return out, nil
}

// AddAgent persists a new agent row and returns the model.
func AddAgent(ctx context.Context, agentURL, username, password string) (*AgentModel, error) {
	id, err := db.InsertAgent(ctx, agentURL, username, password)
	if err != nil {
		return nil, err
	}
	return &AgentModel{Agent: db.Agent{
		ID:       id,
		URL:      agentURL,
		Username: username,
		Password: password,
		Active:   true,
	}}, nil
}

// RemoveAgent deletes an agent by URL and returns the deleted record.
func RemoveAgent(ctx context.Context, agentURL string) (*AgentModel, error) {
	a, err := db.DeleteAgentByURL(ctx, agentURL)
	if err != nil {
		return nil, err
	}
	return &AgentModel{Agent: *a}, nil
}
