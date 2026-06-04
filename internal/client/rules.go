package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	searchRulesPath  = "/north/v80/rules/search"
	rulesCatalogPath = "/north/v80/rules/catalog"
	rulesPath        = "/north/v80/rules/provision/organizations/%s/channels/%s"
	rulePath         = "/north/v80/rules/provision/organizations/%s/channels/%s/%s"
)

// RuleSummary extracts key fields from a rule for display.
type RuleSummary struct {
	Identifier  string          `json:"identifier,omitempty"`
	Name        string          `json:"name,omitempty"`
	Mode        string          `json:"mode,omitempty"` // EASY | ADVANCED
	Active      bool            `json:"active,omitempty"`
	Description string          `json:"description,omitempty"`
	ChannelID   string          `json:"channelId,omitempty"`
	OrgID       string          `json:"organizationId,omitempty"`
	Type        json.RawMessage `json:"type,omitempty"`
}

// RuleTriggerName extracts the trigger type name (DATASTREAM, EVENT, OPERATION)
// from a rule's type field.
func (r RuleSummary) RuleTriggerName() string {
	var t struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(r.Type, &t) == nil {
		return t.Name
	}
	return ""
}

// SearchRulesResponse is the response from the rules search endpoint.
type SearchRulesResponse struct {
	Rules []json.RawMessage `json:"rules"`
	Page  *Page             `json:"page,omitempty"`
}

// ParseRuleSummary extracts display fields from a raw rule.
func ParseRuleSummary(raw json.RawMessage) RuleSummary {
	var s RuleSummary
	_ = json.Unmarshal(raw, &s)
	return s
}

// SearchRules searches for rules using a filter body.
func (c *Client) SearchRules(filter json.RawMessage) (*SearchRulesResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(searchRulesPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search rules: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchRulesResponse{}, nil
	}

	var resp SearchRulesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing rules response: %w", err)
	}
	return &resp, nil
}

// RulesCatalog returns the platform rules catalog (predefined rule templates).
func (c *Client) RulesCatalog() (json.RawMessage, error) {
	data, statusCode, err := c.Get(rulesCatalogPath)
	if err != nil {
		return nil, fmt.Errorf("rules catalog: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// GetRule retrieves a rule by identifier.
func (c *Client) GetRule(org, channel, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(rulePath, org, channel, id)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateRule creates a rule in an organization channel.
func (c *Client) CreateRule(org, channel string, body json.RawMessage) (json.RawMessage, error) {
	path := fmt.Sprintf(rulesPath, org, channel)

	data, statusCode, err := c.Post(path, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateRule updates an existing rule.
func (c *Client) UpdateRule(org, channel, id string, body json.RawMessage) error {
	path := fmt.Sprintf(rulePath, org, channel, id)

	data, statusCode, err := c.Put(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update rule: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteRule deletes a rule.
func (c *Client) DeleteRule(org, channel, id string) error {
	path := fmt.Sprintf(rulePath, org, channel, id)

	data, statusCode, err := c.Delete(path)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// SetRuleActive enables or disables a rule (GET + patch active + PUT).
func (c *Client) SetRuleActive(org, channel, id string, active bool) error {
	raw, err := c.GetRule(org, channel, id)
	if err != nil {
		return err
	}

	var rule map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rule); err != nil {
		return fmt.Errorf("parsing rule: %w", err)
	}
	activeJSON, _ := json.Marshal(active)
	rule["active"] = activeJSON

	body, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("marshaling rule: %w", err)
	}
	return c.UpdateRule(org, channel, id, body)
}
