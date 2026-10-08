package twilio

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Verify service defaults. Any VA SID is accepted and becomes a service
// with these settings unless created explicitly with overrides.
const (
	verifyServicesKey       = "twilio_verify_services"
	defaultFriendlyName     = "mocksms"
	defaultVerifyCodeLength = 6
)

// verifyService is a Verify v2 service record. Services live in the
// project's settings under verifyServicesKey, so no store migration is
// needed; the closed hosted product reads them the same way.
type verifyService struct {
	Sid          string    `json:"sid"`
	FriendlyName string    `json:"friendly_name"`
	CodeLength   int       `json:"code_length"`
	DateCreated  time.Time `json:"date_created"`
	DateUpdated  time.Time `json:"date_updated"`
}

// servicePayload mirrors Twilio's Verify Service resource JSON.
type servicePayload struct {
	Sid          string `json:"sid"`
	FriendlyName string `json:"friendly_name"`
	CodeLength   int    `json:"code_length"`
	AccountSid   string `json:"account_sid"`
	DateCreated  string `json:"date_created"`
	DateUpdated  string `json:"date_updated"`
	URL          string `json:"url"`
}

// verificationPayload mirrors Twilio's Verification resource JSON.
type verificationPayload struct {
	Sid         string  `json:"sid"`
	ServiceSid  string  `json:"service_sid"`
	AccountSid  string  `json:"account_sid"`
	To          string  `json:"to"`
	Channel     string  `json:"channel"`
	Status      string  `json:"status"`
	Valid       bool    `json:"valid"`
	Amount      *string `json:"amount"`
	DateCreated string  `json:"date_created"`
	DateUpdated string  `json:"date_updated"`
	URL         string  `json:"url"`
}

// checkPayload mirrors Twilio's VerificationCheck response JSON.
type checkPayload struct {
	Sid         string  `json:"sid"`
	ServiceSid  string  `json:"service_sid"`
	AccountSid  string  `json:"account_sid"`
	To          string  `json:"to"`
	Channel     string  `json:"channel"`
	Status      string  `json:"status"`
	Valid       bool    `json:"valid"`
	Amount      *string `json:"amount"`
	DateCreated string  `json:"date_created"`
	DateUpdated string  `json:"date_updated"`
}

// isoDate formats a timestamp the way Verify v2 does: ISO 8601 in UTC.
func isoDate(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// serviceURL is the Verify-relative URL of a service resource.
func serviceURL(serviceSid string) string {
	return "/v2/Services/" + serviceSid
}

// verificationURL is the Verify-relative URL of a verification resource.
func verificationURL(serviceSid, verifySid string) string {
	return serviceURL(serviceSid) + "/Verifications/" + verifySid
}

// loadVerifyServices reads the Verify service records for a project.
func (a *Adapter) loadVerifyServices(ctx context.Context, projectID string) (map[string]*verifyService, error) {
	services := map[string]*verifyService{}
	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	raw, ok := project.Settings[verifyServicesKey]
	if !ok {
		return services, nil
	}
	entries, ok := raw.(map[string]interface{})
	if !ok {
		return services, nil
	}
	for sid, entry := range entries {
		fields, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		svc := &verifyService{Sid: sid}
		if v, ok := fields["friendly_name"].(string); ok {
			svc.FriendlyName = v
		}
		switch v := fields["code_length"].(type) {
		case float64:
			svc.CodeLength = int(v)
		case int:
			svc.CodeLength = v
		}
		if v, ok := fields["date_created"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				svc.DateCreated = t
			}
		}
		if v, ok := fields["date_updated"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				svc.DateUpdated = t
			}
		}
		if svc.FriendlyName == "" {
			svc.FriendlyName = defaultFriendlyName
		}
		if svc.CodeLength == 0 {
			svc.CodeLength = defaultVerifyCodeLength
		}
		services[sid] = svc
	}
	return services, nil
}

// saveVerifyService persists a Verify service record into project settings.
func (a *Adapter) saveVerifyService(ctx context.Context, projectID string, svc *verifyService) error {
	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	services, err := a.loadVerifyServices(ctx, projectID)
	if err != nil {
		return err
	}
	services[svc.Sid] = svc
	entries := make(map[string]interface{}, len(services))
	for sid, s := range services {
		entries[sid] = map[string]interface{}{
			"friendly_name": s.FriendlyName,
			"code_length":   s.CodeLength,
			"date_created":  s.DateCreated.UTC().Format(time.RFC3339),
			"date_updated":  s.DateUpdated.UTC().Format(time.RFC3339),
		}
	}
	if project.Settings == nil {
		project.Settings = map[string]interface{}{}
	}
	project.Settings[verifyServicesKey] = entries
	return a.service.Store().UpdateProject(ctx, project)
}

// getOrProvisionService returns the service for a VA SID, auto-provisioning
// defaults for SIDs the SDK uses without creating them first (spec §7.4).
func (a *Adapter) getOrProvisionService(ctx context.Context, projectID, serviceSid string) (*verifyService, error) {
	services, err := a.loadVerifyServices(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if svc, ok := services[serviceSid]; ok {
		return svc, nil
	}
	now := time.Now()
	svc := &verifyService{
		Sid:          serviceSid,
		FriendlyName: defaultFriendlyName,
		CodeLength:   defaultVerifyCodeLength,
		DateCreated:  now,
		DateUpdated:  now,
	}
	if err := a.saveVerifyService(ctx, projectID, svc); err != nil {
		return nil, err
	}
	return svc, nil
}

// renderService converts a service record into Twilio's shape.
func renderService(accountSid string, svc *verifyService) servicePayload {
	return servicePayload{
		Sid:          svc.Sid,
		FriendlyName: svc.FriendlyName,
		CodeLength:   svc.CodeLength,
		AccountSid:   accountSid,
		DateCreated:  isoDate(svc.DateCreated),
		DateUpdated:  isoDate(svc.DateUpdated),
		URL:          serviceURL(svc.Sid),
	}
}

// twilioVerifyStatus maps a canonical verification status onto Verify v2's
// vocabulary. They match except max_attempts, which Twilio spells out.
func twilioVerifyStatus(s core.VerificationStatus) string {
	if s == core.VerificationMaxAttempts {
		return "max_attempts_reached"
	}
	return string(s)
}

// renderVerification converts a core verification into Twilio's shape.
func renderVerification(accountSid string, v *core.Verification) verificationPayload {
	serviceSid := ""
	if v.ServiceRef != nil {
		serviceSid = *v.ServiceRef
	}
	return verificationPayload{
		Sid:         v.ProviderRef,
		ServiceSid:  serviceSid,
		AccountSid:  accountSid,
		To:          v.To,
		Channel:     string(v.Channel),
		Status:      twilioVerifyStatus(v.Status),
		Valid:       v.Status == core.VerificationApproved,
		Amount:      nil,
		DateCreated: isoDate(v.CreatedAt),
		DateUpdated: isoDate(v.CreatedAt),
		URL:         verificationURL(serviceSid, v.ProviderRef),
	}
}

// createService handles POST /v2/Services.
func (a *Adapter) createService(w http.ResponseWriter, r *http.Request) {
	accountSid := adapterkit.Credential(r)
	if err := r.ParseForm(); err != nil {
		a.writeVerifyError(w, core.NewValidationError("invalid form encoding", ""))
		return
	}

	friendlyName := strings.TrimSpace(r.FormValue("FriendlyName"))
	if friendlyName == "" {
		friendlyName = defaultFriendlyName
	}
	codeLength := defaultVerifyCodeLength
	if raw := strings.TrimSpace(r.FormValue("CodeLength")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 4 || v > 8 {
			a.writeVerifyError(w, core.NewValidationError("code_length must be between 4 and 8", "code_length"))
			return
		}
		codeLength = v
	}

	sid, err := newSID("VA")
	if err != nil {
		a.writeVerifyError(w, core.NewInternal("failed to allocate service sid"))
		return
	}
	now := time.Now()
	svc := &verifyService{
		Sid:          sid,
		FriendlyName: friendlyName,
		CodeLength:   codeLength,
		DateCreated:  now,
		DateUpdated:  now,
	}
	if err := a.saveVerifyService(r.Context(), adapterkit.ProjectID(r), svc); err != nil {
		a.writeVerifyError(w, core.NewInternal("failed to store service: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(renderService(accountSid, svc))
}

// fetchService handles GET /v2/Services/{ServiceSid}.
func (a *Adapter) fetchService(w http.ResponseWriter, r *http.Request) {
	serviceSid := chi.URLParam(r, "ServiceSid")
	services, err := a.loadVerifyServices(r.Context(), adapterkit.ProjectID(r))
	if err != nil {
		a.writeVerifyError(w, core.NewInternal("failed to load services: "+err.Error()))
		return
	}
	svc, ok := services[serviceSid]
	if !ok {
		a.writeVerifyError(w, core.NewNotFound("service not found", "sid"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(renderService(adapterkit.Credential(r), svc))
}

// verifyChannel maps a Verify channel name onto the core channel.
func verifyChannel(raw string) (core.Channel, *core.Error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "sms":
		return core.ChannelSMS, nil
	case "email":
		return core.ChannelEmail, nil
	default:
		return "", core.NewValidationError("channel must be sms or email", "channel")
	}
}

// createVerification handles POST /v2/Services/{ServiceSid}/Verifications.
func (a *Adapter) createVerification(w http.ResponseWriter, r *http.Request) {
	serviceSid := chi.URLParam(r, "ServiceSid")
	projectID := adapterkit.ProjectID(r)
	if err := r.ParseForm(); err != nil {
		a.writeVerifyError(w, core.NewValidationError("invalid form encoding", ""))
		return
	}

	to := strings.TrimSpace(r.FormValue("To"))
	if to == "" {
		a.writeVerifyError(w, core.NewValidationError("A 'To' destination is required.", "to"))
		return
	}
	channel, verr := verifyChannel(r.FormValue("Channel"))
	if verr != nil {
		a.writeVerifyError(w, verr)
		return
	}

	svc, err := a.getOrProvisionService(r.Context(), projectID, serviceSid)
	if err != nil {
		a.writeVerifyError(w, core.NewInternal("failed to load service: "+err.Error()))
		return
	}

	sid, err := newSID("VE")
	if err != nil {
		a.writeVerifyError(w, core.NewInternal("failed to allocate verification sid"))
		return
	}
	label := svc.FriendlyName
	resp, err := a.service.StartVerification(r.Context(), projectID, core.VerificationRequest{
		To:           to,
		Channel:      channel,
		CodeLength:   svc.CodeLength,
		Provider:     a.Name(),
		ProviderRef:  sid,
		ServiceRef:   &serviceSid,
		ServiceLabel: &label,
	})
	if err != nil {
		a.writeVerifyServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(renderVerification(adapterkit.Credential(r), resp.Verification))
}

// lookupVerification resolves a VE SID to its core verification, scoped to
// the project and service in the URL.
func (a *Adapter) lookupVerification(ctx context.Context, projectID, serviceSid, verifySid string) (*core.Verification, *core.Error) {
	v, err := a.service.Store().GetVerificationByProviderRef(ctx, projectID, verifySid)
	if err != nil {
		return nil, core.NewNotFound("verification not found", "sid")
	}
	if v.ServiceRef == nil || *v.ServiceRef != serviceSid {
		return nil, core.NewNotFound("verification not found", "sid")
	}
	return v, nil
}

// fetchVerification handles GET /v2/Services/{ServiceSid}/Verifications/{VerificationSid}.
func (a *Adapter) fetchVerification(w http.ResponseWriter, r *http.Request) {
	v, verr := a.lookupVerification(
		r.Context(),
		adapterkit.ProjectID(r),
		chi.URLParam(r, "ServiceSid"),
		chi.URLParam(r, "VerificationSid"),
	)
	if verr != nil {
		a.writeVerifyError(w, verr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(renderVerification(adapterkit.Credential(r), v))
}

// updateVerification handles POST
// /v2/Services/{ServiceSid}/Verifications/{VerificationSid} with
// Status=canceled|approved.
func (a *Adapter) updateVerification(w http.ResponseWriter, r *http.Request) {
	projectID := adapterkit.ProjectID(r)
	serviceSid := chi.URLParam(r, "ServiceSid")
	if err := r.ParseForm(); err != nil {
		a.writeVerifyError(w, core.NewValidationError("invalid form encoding", ""))
		return
	}

	var status core.VerificationStatus
	switch strings.ToLower(strings.TrimSpace(r.FormValue("Status"))) {
	case "canceled":
		status = core.VerificationCanceled
	case "approved":
		status = core.VerificationApproved
	default:
		a.writeVerifyError(w, core.NewValidationError("status must be canceled or approved", "status"))
		return
	}

	v, verr := a.lookupVerification(r.Context(), projectID, serviceSid, chi.URLParam(r, "VerificationSid"))
	if verr != nil {
		a.writeVerifyError(w, verr)
		return
	}
	if v.Status != core.VerificationPending {
		a.writeVerifyError(w, core.NewValidationError("verification is "+string(v.Status), "status"))
		return
	}

	v.Status = status
	if err := a.service.Store().UpdateVerification(r.Context(), v); err != nil {
		a.writeVerifyError(w, core.NewInternal("failed to update verification: "+err.Error()))
		return
	}
	a.service.Bus().Publish(r.Context(), core.Event{
		Type:      core.EventVerificationUpdated,
		Payload:   v,
		ProjectID: projectID,
		Timestamp: a.service.Clock().Now(),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(renderVerification(adapterkit.Credential(r), v))
}

// findPendingVerification returns the newest pending verification for a
// service and recipient, which is what a To-based check targets.
func (a *Adapter) findPendingVerification(ctx context.Context, projectID, serviceSid, to string) *core.Verification {
	verifications, _, err := a.service.Store().ListVerifications(ctx, projectID, maxScanLimit, "")
	if err != nil {
		return nil
	}
	var newest *core.Verification
	for _, v := range verifications {
		if v.Provider != a.Name() || v.Status != core.VerificationPending {
			continue
		}
		if v.ServiceRef == nil || *v.ServiceRef != serviceSid {
			continue
		}
		if v.To != to {
			continue
		}
		if newest == nil || v.CreatedAt.After(newest.CreatedAt) {
			newest = v
		}
	}
	return newest
}

// checkVerification handles POST /v2/Services/{ServiceSid}/VerificationCheck
// addressed by To or by VerificationSid.
func (a *Adapter) checkVerification(w http.ResponseWriter, r *http.Request) {
	projectID := adapterkit.ProjectID(r)
	serviceSid := chi.URLParam(r, "ServiceSid")
	if err := r.ParseForm(); err != nil {
		a.writeVerifyError(w, core.NewValidationError("invalid form encoding", ""))
		return
	}

	code := strings.TrimSpace(r.FormValue("Code"))
	if code == "" {
		a.writeVerifyError(w, core.NewValidationError("A 'Code' is required.", "code"))
		return
	}

	var verificationID string
	if sid := strings.TrimSpace(r.FormValue("VerificationSid")); sid != "" {
		v, verr := a.lookupVerification(r.Context(), projectID, serviceSid, sid)
		if verr != nil {
			a.writeVerifyError(w, verr)
			return
		}
		verificationID = v.ID
	} else if to := strings.TrimSpace(r.FormValue("To")); to != "" {
		channel, verr := verifyChannel(r.FormValue("Channel"))
		if verr != nil {
			a.writeVerifyError(w, verr)
			return
		}
		lookup := to
		if channel == core.ChannelSMS {
			lookup = core.NormalizePhone(to)
		}
		v := a.findPendingVerification(r.Context(), projectID, serviceSid, lookup)
		if v == nil {
			a.writeVerifyError(w, core.NewNotFound("no pending verification for recipient", "to"))
			return
		}
		verificationID = v.ID
	} else {
		a.writeVerifyError(w, core.NewValidationError("A 'To' or 'VerificationSid' is required.", "to"))
		return
	}

	if _, err := a.service.CheckVerification(r.Context(), projectID, verificationID, core.CheckVerificationRequest{Code: code}); err != nil {
		a.writeVerifyServiceError(w, err)
		return
	}

	v, verr := a.service.Store().GetVerification(r.Context(), projectID, verificationID)
	if verr != nil {
		a.writeVerifyError(w, core.NewInternal("failed to load verification: "+verr.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(checkPayload{
		Sid:         v.ProviderRef,
		ServiceSid:  serviceSid,
		AccountSid:  adapterkit.Credential(r),
		To:          v.To,
		Channel:     string(v.Channel),
		Status:      twilioVerifyStatus(v.Status),
		Valid:       v.Status == core.VerificationApproved,
		Amount:      nil,
		DateCreated: isoDate(v.CreatedAt),
		DateUpdated: isoDate(v.CreatedAt),
	})
}
