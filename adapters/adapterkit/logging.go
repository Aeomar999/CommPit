package adapterkit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

// DefaultMaxBodySize defines the maximum number of bytes recorded in RequestLog
// for request and response payloads (64 KB).
const DefaultMaxBodySize = 64 * 1024

// RequestLogSink receives finalized RequestLog entries for persistent storage.
type RequestLogSink interface {
	RecordRequestLog(ctx context.Context, entry *core.RequestLog) error
}

// RequestLogSinkFunc allows using a standalone function as a RequestLogSink.
type RequestLogSinkFunc func(ctx context.Context, entry *core.RequestLog) error

// RecordRequestLog executes the underlying function.
func (f RequestLogSinkFunc) RecordRequestLog(ctx context.Context, entry *core.RequestLog) error {
	return f(ctx, entry)
}

// LoggingConfig defines configuration parameters for request auditing.
type LoggingConfig struct {
	AdapterName string
	Sink        RequestLogSink
	Bus         core.Bus
	Clock       core.Clock
	MaxBodySize int
}

type responseCapture struct {
	http.ResponseWriter
	status      int
	body        bytes.Buffer
	maxBodySize int
	written     int64
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	if rc.status == 0 {
		rc.status = statusCode
		rc.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rc *responseCapture) Write(buf []byte) (int, error) {
	if rc.status == 0 {
		rc.status = http.StatusOK
	}
	if rc.body.Len() < rc.maxBodySize {
		available := rc.maxBodySize - rc.body.Len()
		if len(buf) <= available {
			rc.body.Write(buf)
		} else {
			rc.body.Write(buf[:available])
		}
	}
	rc.written += int64(len(buf))
	return rc.ResponseWriter.Write(buf)
}

func (rc *responseCapture) Unwrap() http.ResponseWriter {
	return rc.ResponseWriter
}

func (rc *responseCapture) Flush() {
	if flusher, ok := rc.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// RequestLogger wraps downstream handlers to audit HTTP transactions into core.RequestLog
// entries, masking credentials and enforcing a 64 KB size cap on recorded payloads.
func RequestLogger(adapter Adapter, cfg LoggingConfig) func(http.Handler) http.Handler {
	maxCap := cfg.MaxBodySize
	if maxCap <= 0 {
		maxCap = DefaultMaxBodySize
	}
	clock := cfg.Clock
	if clock == nil {
		clock = core.RealClock{}
	}
	adapterName := cfg.AdapterName
	if adapterName == "" && adapter != nil {
		adapterName = adapter.Name()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			ctx := EnsureCarrier(req.Context())
			req = req.WithContext(ctx)

			startTime := clock.Now()
			capturedReq := captureRequestBody(req, maxCap)
			rawHeaders := extractHeaders(req)

			rc := &responseCapture{
				ResponseWriter: writer,
				maxBodySize:    maxCap,
			}

			next.ServeHTTP(rc, req)

			duration := clock.Now().Sub(startTime).Milliseconds()
			logEntry := buildRequestLog(req, rc, capturedReq, rawHeaders, adapterName, maxCap, duration, startTime)
			emitRequestLog(logEntry, cfg.Sink, cfg.Bus)
		})
	}
}

func captureRequestBody(req *http.Request, maxCap int) []byte {
	if req.Body == nil {
		return nil
	}
	limited := io.LimitReader(req.Body, int64(maxCap))
	buf, _ := io.ReadAll(limited)
	req.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf), req.Body))
	return buf
}

func extractHeaders(req *http.Request) map[string]string {
	result := make(map[string]string, len(req.Header))
	for key, values := range req.Header {
		if len(values) > 0 {
			result[key] = strings.Join(values, ", ")
		}
	}
	return result
}

func buildRequestLog(
	req *http.Request,
	rc *responseCapture,
	capturedReq []byte,
	rawHeaders map[string]string,
	adapterName string,
	maxCap int,
	duration int64,
	startTime time.Time,
) *core.RequestLog {
	status := rc.status
	if status == 0 {
		status = http.StatusOK
	}

	contentType := req.Header.Get("Content-Type")
	maskedReq := MaskBody(contentType, capturedReq)
	if len(maskedReq) > maxCap {
		maskedReq = maskedReq[:maxCap]
	}

	respContentType := rc.Header().Get("Content-Type")
	maskedResp := MaskBody(respContentType, rc.body.Bytes())
	if len(maskedResp) > maxCap {
		maskedResp = maskedResp[:maxCap]
	}

	uri := req.URL.RequestURI()
	if uri == "" {
		uri = req.URL.Path
	}

	return &core.RequestLog{
		ID:             core.NewRequestLogID(),
		ProjectID:      ProjectIDFromContext(req.Context()),
		Adapter:        adapterName,
		Method:         req.Method,
		Path:           MaskURL(uri),
		RequestHeaders: MaskHeaders(rawHeaders),
		RequestBody:    maskedReq,
		ResponseStatus: status,
		ResponseBody:   maskedResp,
		DurationMS:     duration,
		CreatedAt:      startTime,
	}
}

func emitRequestLog(entry *core.RequestLog, sink RequestLogSink, bus core.Bus) {
	bgCtx := context.Background()
	if sink != nil {
		_ = sink.RecordRequestLog(bgCtx, entry)
	}
	if bus != nil {
		bus.Publish(bgCtx, core.Event{
			Type:      core.EventRequestLogged,
			ProjectID: entry.ProjectID,
			Payload:   entry,
			Timestamp: entry.CreatedAt,
		})
	}
}
