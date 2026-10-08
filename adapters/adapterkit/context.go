package adapterkit

import (
	"context"
	"net/http"
)

type contextKey int

const (
	carrierKey contextKey = iota
)

type contextCarrier struct {
	projectID  string
	credential string
}

// EnsureCarrier attaches a contextCarrier to ctx if one is not already present.
func EnsureCarrier(ctx context.Context) context.Context {
	if _, ok := ctx.Value(carrierKey).(*contextCarrier); ok {
		return ctx
	}
	carrier := &contextCarrier{
		projectID: "default",
	}
	return context.WithValue(ctx, carrierKey, carrier)
}

// WithProjectID associates a project ID with the context.
func WithProjectID(ctx context.Context, projectID string) context.Context {
	if carrier, ok := ctx.Value(carrierKey).(*contextCarrier); ok {
		carrier.projectID = projectID
		return ctx
	}
	carrier := &contextCarrier{projectID: projectID}
	return context.WithValue(ctx, carrierKey, carrier)
}

// ProjectID extracts the resolved project ID from the incoming HTTP request.
func ProjectID(req *http.Request) string {
	return ProjectIDFromContext(req.Context())
}

// ProjectIDFromContext extracts the resolved project ID from the context.
// Returns "default" if no specific project was attached.
func ProjectIDFromContext(ctx context.Context) string {
	if carrier, ok := ctx.Value(carrierKey).(*contextCarrier); ok && carrier.projectID != "" {
		return carrier.projectID
	}
	return "default"
}

// WithCredential associates an extracted credential identifier with the context.
func WithCredential(ctx context.Context, credential string) context.Context {
	if carrier, ok := ctx.Value(carrierKey).(*contextCarrier); ok {
		carrier.credential = credential
		return ctx
	}
	carrier := &contextCarrier{credential: credential}
	return context.WithValue(ctx, carrierKey, carrier)
}

// Credential extracts the caller's credential identifier from the incoming HTTP request.
func Credential(req *http.Request) string {
	return CredentialFromContext(req.Context())
}

// CredentialFromContext extracts the caller's credential identifier from the context.
func CredentialFromContext(ctx context.Context) string {
	if carrier, ok := ctx.Value(carrierKey).(*contextCarrier); ok {
		return carrier.credential
	}
	return ""
}
