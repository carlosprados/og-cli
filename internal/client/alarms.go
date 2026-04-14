package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	searchAlarmsPath  = "/north/v80/search/entities/alarms"
	summaryAlarmsPath = "/north/v80/search/entities/alarms/summary"
	alarmsActionPath  = "/north/v80/alarms"
)

// Alarm represents an OpenGate alarm instance.
type Alarm struct {
	Identifier          string `json:"identifier"`
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	Severity            string `json:"severity"`
	Status              string `json:"status"`
	Priority            string `json:"priority,omitempty"`
	Rule                string `json:"rule,omitempty"`
	Organization        string `json:"organization,omitempty"`
	Channel             string `json:"channel,omitempty"`
	EntityIdentifier    string `json:"entityIdentifier,omitempty"`
	SubEntityIdentifier string `json:"subEntityIdentifier,omitempty"`
	ResourceType        string `json:"resourceType,omitempty"`
	OpeningDate         string `json:"openingDate,omitempty"`
	AttentionDate       string `json:"attentionDate,omitempty"`
	ClosureDate         string `json:"closureDate,omitempty"`
	AttentionUser       string `json:"attentionUser,omitempty"`
	AttentionNote       string `json:"attentionNote,omitempty"`
	ClosureUser         string `json:"closureUser,omitempty"`
	ClosureNote         string `json:"closureNote,omitempty"`
}

// SearchAlarmsResponse is the response from the alarms search endpoint.
type SearchAlarmsResponse struct {
	Alarms []Alarm `json:"alarms"`
	Page   *Page   `json:"page,omitempty"`
}

// AlarmSummaryGroup is a single group entry in a summary.
type AlarmSummaryGroup struct {
	Count int                  `json:"count"`
	List  []AlarmSummaryEntry  `json:"list"`
}

// AlarmSummaryEntry is a name/count pair within a summary group.
type AlarmSummaryEntry struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// AlarmSummary is the response from the alarms summary endpoint.
type AlarmSummary struct {
	Date         string                       `json:"date"`
	Count        int                          `json:"count"`
	SummaryGroup []map[string]AlarmSummaryGroup `json:"summaryGroup"`
}

// AlarmSummaryResponse wraps the summary.
type AlarmSummaryResponse struct {
	Summary AlarmSummary `json:"summary"`
}

// AlarmActionRequest is the body for attend/close operations.
type AlarmActionRequest struct {
	Action string   `json:"action"`
	Alarms []string `json:"alarms"`
	Notes  string   `json:"notes,omitempty"`
}

// AlarmActionResponse is the response from an alarm action.
type AlarmActionResponse struct {
	Result struct {
		Count      int `json:"count"`
		Successful int `json:"succesfull"`
		Error      struct {
			Count    int `json:"count"`
			NotExist struct {
				Count int      `json:"count"`
				List  []string `json:"list"`
			} `json:"notExist"`
		} `json:"error"`
	} `json:"result"`
}

// SearchAlarms searches for alarms using a filter body.
func (c *Client) SearchAlarms(filter json.RawMessage) (*SearchAlarmsResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(searchAlarmsPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search alarms: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	if IsEmptyResponse(data, statusCode) {
		return &SearchAlarmsResponse{}, nil
	}

	var resp SearchAlarmsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing alarms response: %w", err)
	}
	return &resp, nil
}

// SummaryAlarms returns a summary of alarms grouped by severity, status, rule, name.
func (c *Client) SummaryAlarms(filter json.RawMessage) (*AlarmSummaryResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(summaryAlarmsPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("alarms summary: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	if IsEmptyResponse(data, statusCode) {
		return &AlarmSummaryResponse{}, nil
	}

	var resp AlarmSummaryResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing alarms summary: %w", err)
	}
	return &resp, nil
}

// AttendAlarms marks alarms as attended.
func (c *Client) AttendAlarms(ids []string, notes string) (*AlarmActionResponse, error) {
	return c.alarmAction("ATTEND", ids, notes)
}

// CloseAlarms marks alarms as closed.
func (c *Client) CloseAlarms(ids []string, notes string) (*AlarmActionResponse, error) {
	return c.alarmAction("CLOSE", ids, notes)
}

func (c *Client) alarmAction(action string, ids []string, notes string) (*AlarmActionResponse, error) {
	req := AlarmActionRequest{
		Action: action,
		Alarms: ids,
		Notes:  notes,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling alarm action: %w", err)
	}

	data, statusCode, err := c.Post(alarmsActionPath, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("alarm %s: %w", action, err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var resp AlarmActionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing alarm action response: %w", err)
	}
	return &resp, nil
}
