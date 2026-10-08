package smtpd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/Aeomar999/CommPit/core"
	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
)

type Server struct {
	cfg            *Config
	service        *core.Service
	resolver       core.ProjectResolver
	listener       net.Listener
	server         *smtp.Server
	maxMessageSize int64
}

type Config struct {
	Host           string
	Port           int
	MaxMessageSize int64
	EnableSTARTTLS bool
	TLSConfig      *tls.Config
}

func NewServer(cfg *Config, service *core.Service, resolver core.ProjectResolver) *Server {
	s := &Server{
		cfg:            cfg,
		service:        service,
		resolver:       resolver,
		maxMessageSize: cfg.MaxMessageSize,
	}

	smtpServer := smtp.NewServer(&SMTPBackend{
		service:  service,
		resolver: resolver,
		maxSize:  cfg.MaxMessageSize,
	})

	smtpServer.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	smtpServer.Domain = "localhost"
	smtpServer.ReadTimeout = 30 * time.Second
	smtpServer.WriteTimeout = 30 * time.Second
	smtpServer.MaxMessageBytes = cfg.MaxMessageSize
	smtpServer.MaxRecipients = 100
	smtpServer.AllowInsecureAuth = true

	if cfg.EnableSTARTTLS && cfg.TLSConfig != nil {
		smtpServer.TLSConfig = cfg.TLSConfig
	}

	s.server = smtpServer
	return s
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.server.Addr, err)
	}
	s.listener = ln
	slog.Info("SMTP server listening", "addr", s.server.Addr)
	return s.server.Serve(ln)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// SMTPBackend implements smtp.Backend
type SMTPBackend struct {
	service  *core.Service
	resolver core.ProjectResolver
	maxSize  int64
}

func (b *SMTPBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{
		backend: b,
		conn:    c,
	}, nil
}

// Session implements smtp.Session
type Session struct {
	backend   *SMTPBackend
	conn      *smtp.Conn
	from      string
	to        []string
	data      []byte
	authUser  string
	projectID string
}

func (s *Session) AuthPlain(username, password string) error {
	projectID, err := s.backend.resolver.Resolve(context.Background(), "smtp", username)
	if err != nil {
		return errors.New("authentication failed")
	}
	s.authUser = username
	s.projectID = projectID
	return nil
}

func (s *Session) AuthLogin(username, password string) error {
	return s.AuthPlain(username, password)
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if opts != nil && opts.Size > s.backend.maxSize {
		return errors.New("message size exceeds limit")
	}
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// Read message data with size limit
	limitedReader := io.LimitReader(r, s.backend.maxSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return err
	}
	s.data = data
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func (s *Session) Submit() error {
	if s.projectID == "" {
		// Try to resolve project from auth user
		if s.authUser != "" {
			projectID, err := s.backend.resolver.Resolve(context.Background(), "smtp", s.authUser)
			if err == nil {
				s.projectID = projectID
			}
		}
	}

	if s.projectID == "" {
		return errors.New("no project ID available (authentication required)")
	}

	if len(s.to) == 0 {
		return errors.New("no recipients")
	}

	// Parse MIME message
	env, parseErr := enmime.ReadEnvelope(strings.NewReader(string(s.data)))
	if parseErr != nil {
		return fmt.Errorf("failed to parse MIME: %w", parseErr)
	}

	// Extract headers
	subject := env.GetHeader("Subject")
	bodyText := env.Text
	bodyHTML := env.HTML

	// Extract attachments
	var attachments []core.AttachmentInput
	for _, part := range env.Attachments {
		attachments = append(attachments, core.AttachmentInput{
			Filename:      part.FileName,
			ContentType:   part.ContentType,
			ContentBase64: "",
			InlineCID:     part.ContentID,
		})
	}

	// Use the first recipient for the message
	to := s.to[0]

	sendReq := core.SendRequest{
		Channel:     core.ChannelEmail,
		From:        s.from,
		To:          []string{to},
		Subject:     subject,
		BodyText:    bodyText,
		BodyHTML:    bodyHTML,
		Attachments: attachments,
		Provider:    "smtp",
	}

	_, err := s.backend.service.SendMessage(context.Background(), s.projectID, sendReq)
	return err
}
