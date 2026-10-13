// Package mcpserver exposes core messaging operations as MCP tools over
// streamable HTTP (spec §8.7). Each tool calls the core service directly:
// no HTTP loopback, no provider formats.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Aeomar999/CommPit/core"
)

// Server is the MCP server around the core service.
type Server struct {
	service *core.Service
	mcp     *mcp.Server
}

// New wires the seven tools and returns the server.
func New(service *core.Service) *Server {
	s := &Server{service: service}
	ms := mcp.NewServer(&mcp.Implementation{Name: "mocksms", Version: "0.4.0"}, nil)
	mcp.AddTool(ms, &mcp.Tool{Name: "list_messages", Description: "List messages in a project, newest first."}, s.listMessages)
	mcp.AddTool(ms, &mcp.Tool{Name: "wait_for_message", Description: "Wait for a message to arrive, then return it."}, s.waitForMessage)
	mcp.AddTool(ms, &mcp.Tool{Name: "get_latest_otp", Description: "Newest OTP code for a recipient."}, s.getLatestOTP)
	mcp.AddTool(ms, &mcp.Tool{Name: "get_latest_email", Description: "Newest email for a recipient."}, s.getLatestEmail)
	mcp.AddTool(ms, &mcp.Tool{Name: "simulate_inbound", Description: "Simulate an inbound SMS from a handset."}, s.simulateInbound)
	mcp.AddTool(ms, &mcp.Tool{Name: "list_requests", Description: "List provider-adapter request logs."}, s.listRequests)
	mcp.AddTool(ms, &mcp.Tool{Name: "reset_project", Description: "Delete every message in a project."}, s.resetProject)
	s.mcp = ms
	return s
}

// Handler serves streamable HTTP for mounting at /mcp.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.mcp }, nil)
}

// resolveProject maps an optional project argument to a project ID. Empty
// means the default project; anything else must be an existing ULID.
func (s *Server) resolveProject(ctx context.Context, project string) (string, error) {
	if project == "" {
		return s.service.Resolver().Resolve(ctx, "native", "default")
	}
	if _, err := s.service.Store().GetProject(ctx, project); err != nil {
		return "", err
	}
	return project, nil
}

func textResult(v any) (*mcp.CallToolResult, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil
}

type listMessagesInput struct {
	Project   string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
	Channel   string `json:"channel,omitempty" jsonschema:"sms or email"`
	To        string `json:"to,omitempty" jsonschema:"Filter by recipient"`
	From      string `json:"from,omitempty" jsonschema:"Filter by sender"`
	Status    string `json:"status,omitempty" jsonschema:"Filter by status"`
	Direction string `json:"direction,omitempty" jsonschema:"outbound or inbound"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Max rows, default 20"`
}

func (s *Server) listMessages(ctx context.Context, _ *mcp.CallToolRequest, in listMessagesInput) (*mcp.CallToolResult, any, error) {
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	filter := core.MessageFilter{Limit: in.Limit}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if in.Channel != "" {
		channel := core.Channel(in.Channel)
		filter.Channel = &channel
	}
	if in.To != "" {
		filter.To = &in.To
	}
	if in.From != "" {
		filter.From = &in.From
	}
	if in.Status != "" {
		status := core.MessageStatus(in.Status)
		filter.Status = &status
	}
	if in.Direction != "" {
		direction := core.Direction(in.Direction)
		filter.Direction = &direction
	}
	messages, _, err := s.service.Store().ListMessages(ctx, projectID, filter)
	if err != nil {
		return nil, nil, err
	}
	res, err := textResult(messages)
	return res, messages, err
}

type waitForMessageInput struct {
	Project        string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
	To             string `json:"to,omitempty" jsonschema:"Only match this recipient"`
	Channel        string `json:"channel,omitempty" jsonschema:"sms or email"`
	Since          string `json:"since,omitempty" jsonschema:"Only match messages after this RFC3339 time; default call time"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"Wait budget, default 10, max 60"`
}

func (s *Server) waitForMessage(ctx context.Context, _ *mcp.CallToolRequest, in waitForMessageInput) (*mcp.CallToolResult, any, error) {
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	timeout := time.Duration(in.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if timeout > 60*time.Second {
		timeout = 60 * time.Second
	}
	var channel *core.Channel
	if in.Channel != "" {
		c := core.Channel(in.Channel)
		channel = &c
	}
	since := s.service.Clock().Now()
	if in.Since != "" {
		if t, err := time.Parse(time.RFC3339, in.Since); err == nil {
			since = t
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	messages, _, err := s.service.Store().ListMessages(ctx, projectID, core.MessageFilter{
		To:      emptyToNil(in.To),
		Channel: channel,
		Since:   &since,
		Limit:   1,
	})
	if err == nil && len(messages) > 0 {
		res, err := textResult(messages[0])
		return res, messages[0], err
	}

	msgCh := make(chan *core.Message, 1)
	sub := s.service.Bus().Subscribe(string(core.EventMessageCreated), func(e core.Event) {
		msg, ok := e.Payload.(*core.Message)
		if !ok || msg.ProjectID != projectID {
			return
		}
		if in.To != "" && msg.To != in.To {
			return
		}
		if channel != nil && msg.Channel != *channel {
			return
		}
		if !msg.CreatedAt.After(since) {
			return
		}
		select {
		case msgCh <- msg:
		default:
		}
	})
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("mcpserver: wait timeout")
	case msg := <-msgCh:
		res, err := textResult(msg)
		return res, msg, err
	}
}

func emptyToNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type latestOTPInput struct {
	Project string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
	To      string `json:"to" jsonschema:"Recipient to look up (required)"`
}

type latestOTPOutput struct {
	Code           string  `json:"code"`
	Source         string  `json:"source"`
	MessageID      string  `json:"message_id"`
	VerificationID *string `json:"verification_id,omitempty"`
}

func (s *Server) getLatestOTP(ctx context.Context, _ *mcp.CallToolRequest, in latestOTPInput) (*mcp.CallToolResult, any, error) {
	if in.To == "" {
		return nil, nil, fmt.Errorf("mcpserver: to is required")
	}
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	normalized := core.NormalizePhone(in.To)
	direction := core.DirectionOutbound
	messages, _, err := s.service.Store().ListMessages(ctx, projectID, core.MessageFilter{
		To:        &normalized,
		Direction: &direction,
		Limit:     100,
	})
	if err != nil {
		return nil, nil, err
	}
	var latest *core.Message
	for _, m := range messages {
		if len(m.ExtractedCodes) > 0 {
			latest = m
			break
		}
	}
	if latest == nil {
		return nil, nil, fmt.Errorf("mcpserver: no OTP found for %s", in.To)
	}
	out := latestOTPOutput{Code: latest.ExtractedCodes[0], Source: "extracted", MessageID: latest.ID}
	verifications, _, err := s.service.Store().ListVerifications(ctx, projectID, 1000, "")
	if err != nil {
		return nil, nil, err
	}
	for _, v := range verifications {
		if v.MessageID == latest.ID {
			out.Code = v.Code
			out.Source = "verification"
			out.VerificationID = &v.ID
			break
		}
	}
	res, err := textResult(out)
	return res, out, err
}

type latestEmailInput struct {
	Project string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
	To      string `json:"to" jsonschema:"Recipient email (required)"`
}

func (s *Server) getLatestEmail(ctx context.Context, _ *mcp.CallToolRequest, in latestEmailInput) (*mcp.CallToolResult, any, error) {
	if in.To == "" {
		return nil, nil, fmt.Errorf("mcpserver: to is required")
	}
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	channel := core.ChannelEmail
	direction := core.DirectionOutbound
	messages, _, err := s.service.Store().ListMessages(ctx, projectID, core.MessageFilter{
		Channel:   &channel,
		To:        &in.To,
		Direction: &direction,
		Limit:     1,
	})
	if err != nil {
		return nil, nil, err
	}
	if len(messages) == 0 {
		return nil, nil, fmt.Errorf("mcpserver: no email found for %s", in.To)
	}
	res, err := textResult(messages[0])
	return res, messages[0], err
}

type simulateInboundInput struct {
	Project string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
	From    string `json:"from" jsonschema:"Handset sending the message (required)"`
	To      string `json:"to" jsonschema:"Recipient number (required)"`
	Body    string `json:"body" jsonschema:"Message text (required)"`
}

func (s *Server) simulateInbound(ctx context.Context, _ *mcp.CallToolRequest, in simulateInboundInput) (*mcp.CallToolResult, any, error) {
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	msg, err := s.service.ReceiveInbound(ctx, projectID, core.InboundRequest{
		From: in.From,
		To:   in.To,
		Body: in.Body,
	})
	if err != nil {
		return nil, nil, err
	}
	res, err := textResult(msg)
	return res, msg, err
}

type listRequestsInput struct {
	Project string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max rows, default 20"`
}

func (s *Server) listRequests(ctx context.Context, _ *mcp.CallToolRequest, in listRequestsInput) (*mcp.CallToolResult, any, error) {
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	logs, _, err := s.service.Store().ListRequestLogs(ctx, projectID, limit, "")
	if err != nil {
		return nil, nil, err
	}
	res, err := textResult(logs)
	return res, logs, err
}

type resetProjectInput struct {
	Project string `json:"project,omitempty" jsonschema:"Project ULID; empty uses the default project"`
}

type resetProjectOutput struct {
	Project string `json:"project"`
	Reset   bool   `json:"reset"`
}

func (s *Server) resetProject(ctx context.Context, _ *mcp.CallToolRequest, in resetProjectInput) (*mcp.CallToolResult, any, error) {
	projectID, err := s.resolveProject(ctx, in.Project)
	if err != nil {
		return nil, nil, err
	}
	if err := s.service.Store().DeleteMessages(ctx, projectID); err != nil {
		return nil, nil, err
	}
	out := resetProjectOutput{Project: projectID, Reset: true}
	res, err := textResult(out)
	return res, out, err
}
