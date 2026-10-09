package twilio

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// maxScanLimit bounds the in-memory scans the adapter performs. The core
// store has no provider or provider-ref index, so list and fetch filter in
// memory; totals and paging are exact while a project holds fewer messages
// than this cap.
const maxScanLimit = 5000

// messagePayload mirrors Twilio's Message resource JSON. Twilio returns
// num_segments and num_media as strings and nulls for unpriced messages.
type messagePayload struct {
	Sid                 string            `json:"sid"`
	DateCreated         string            `json:"date_created"`
	DateUpdated         string            `json:"date_updated"`
	DateSent            *string           `json:"date_sent"`
	AccountSid          string            `json:"account_sid"`
	To                  string            `json:"to"`
	From                string            `json:"from"`
	MessagingServiceSid *string           `json:"messaging_service_sid"`
	Body                string            `json:"body"`
	Status              string            `json:"status"`
	NumSegments         string            `json:"num_segments"`
	NumMedia            string            `json:"num_media"`
	Direction           string            `json:"direction"`
	APIVersion          string            `json:"api_version"`
	Price               *string           `json:"price"`
	PriceUnit           string            `json:"price_unit"`
	ErrorCode           *int              `json:"error_code"`
	ErrorMessage        *string           `json:"error_message"`
	URI                 string            `json:"uri"`
	SubresourceURIs     map[string]string `json:"subresource_uris"`
}

// listPayload mirrors Twilio's paging envelope for list responses.
type listPayload struct {
	Messages        []messagePayload `json:"messages"`
	Total           int              `json:"total"`
	Page            int              `json:"page"`
	NumPages        int              `json:"num_pages"`
	PageSize        int              `json:"page_size"`
	Start           int              `json:"start"`
	End             int              `json:"end"`
	FirstPageURI    string           `json:"first_page_uri"`
	NextPageURI     *string          `json:"next_page_uri"`
	PreviousPageURI *string          `json:"previous_page_uri"`
	URI             string           `json:"uri"`
}

// twilioDate formats a timestamp the way Twilio does: RFC 2822 in UTC.
func twilioDate(t time.Time) string {
	return t.UTC().Format(time.RFC1123Z)
}

// twilioStatus maps a canonical message status to Twilio's status vocabulary.
// The vocabularies match one-to-one.
func twilioStatus(s core.MessageStatus) string {
	return string(s)
}

// twilioDirection maps the canonical direction to Twilio's vocabulary.
func twilioDirection(d core.Direction) string {
	if d == core.DirectionInbound {
		return "inbound"
	}
	return "outbound-api"
}

// messageURI is the Twilio-relative URI of a message resource. It omits the
// local /twilio mount prefix so SDKs see the path Twilio would return.
func messageURI(accountSid, messageSid string) string {
	return "/2010-04-01/Accounts/" + accountSid + "/Messages/" + messageSid + ".json"
}

// renderMessage converts a stored core message into Twilio's shape.
func renderMessage(accountSid string, msg *core.Message, numMedia int) messagePayload {
	var dateSent *string
	if msg.Status != core.StatusQueued {
		s := twilioDate(msg.UpdatedAt)
		dateSent = &s
	}
	var errorCode *int
	if msg.ErrorCode != nil {
		if n, err := strconv.Atoi(*msg.ErrorCode); err == nil {
			errorCode = &n
		}
	}
	uri := messageURI(accountSid, msg.ProviderRef)
	return messagePayload{
		Sid:                 msg.ProviderRef,
		DateCreated:         twilioDate(msg.CreatedAt),
		DateUpdated:         twilioDate(msg.UpdatedAt),
		DateSent:            dateSent,
		AccountSid:          accountSid,
		To:                  msg.To,
		From:                msg.From,
		MessagingServiceSid: nil,
		Body:                msg.BodyText,
		Status:              twilioStatus(msg.Status),
		NumSegments:         strconv.Itoa(msg.Segments),
		NumMedia:            strconv.Itoa(numMedia),
		Direction:           twilioDirection(msg.Direction),
		APIVersion:          APIVersion,
		Price:               nil,
		PriceUnit:           "USD",
		ErrorCode:           errorCode,
		ErrorMessage:        msg.ErrorMessage,
		URI:                 uri,
		SubresourceURIs: map[string]string{
			"media": strings.TrimSuffix(uri, ".json") + "/Media.json",
		},
	}
}

// countMediaURLs counts MediaUrl attachments in a submitted form. The URLs
// themselves are accepted but not stored; only the count is reported.
func countMediaURLs(form url.Values) int {
	count := len(form["MediaUrl"])
	for key := range form {
		if len(key) > 8 && strings.HasPrefix(key, "MediaUrl") {
			if _, err := strconv.Atoi(key[8:]); err == nil {
				count += len(form[key])
			}
		}
	}
	return count
}

// createMessage handles POST /2010-04-01/Accounts/{AC}/Messages.json.
func (a *Adapter) createMessage(w http.ResponseWriter, r *http.Request) {
	accountSid := chi.URLParam(r, "AccountSid")
	if err := r.ParseForm(); err != nil {
		a.WriteError(w, core.NewValidationError("invalid form encoding", ""))
		return
	}

	to := strings.TrimSpace(r.FormValue("To"))
	from := strings.TrimSpace(r.FormValue("From"))
	body := r.FormValue("Body")
	statusCallback := strings.TrimSpace(r.FormValue("StatusCallback"))
	numMedia := countMediaURLs(r.Form)

	if user, pass, ok := r.BasicAuth(); ok {
		a.recordCredential(r.Context(), adapterkit.ProjectID(r), user, pass)
	}

	if to == "" {
		a.WriteError(w, core.NewValidationError("A 'To' phone number is required.", "to"))
		return
	}
	if from == "" {
		a.WriteError(w, core.NewValidationError("The 'From' phone number is required.", "from"))
		return
	}
	if body == "" && numMedia == 0 {
		a.WriteError(w, core.NewValidationError("A message body or media URL is required.", "body"))
		return
	}

	sid, err := newSID("SM")
	if err != nil {
		a.WriteError(w, core.NewInternal("failed to allocate message sid"))
		return
	}

	resp, err := a.service.SendMessage(r.Context(), adapterkit.ProjectID(r), core.SendRequest{
		Channel:     core.ChannelSMS,
		From:        from,
		To:          []string{to},
		BodyText:    body,
		CallbackURL: statusCallback,
		Provider:    a.Name(),
		ProviderRef: sid,
	})
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	if resp == nil || resp.Message == nil {
		a.WriteError(w, core.NewInternal("failed to create message"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(renderMessage(accountSid, resp.Message, numMedia))
}

// dateFilter holds the parsed DateSent filters for a list request.
type dateFilter struct {
	exact  *time.Time
	after  *time.Time
	before *time.Time
}

// parseDateFilter reads Twilio's DateSent filters: an exact YYYY-MM-DD date
// plus the DateSent>, DateSent<, DateSent>= and DateSent<= operators.
// Unparsable values are ignored (lenient sandbox behavior).
func parseDateFilter(query url.Values) dateFilter {
	var f dateFilter
	parse := func(v string) *time.Time {
		if v == "" {
			return nil
		}
		if t, err := time.Parse("2006-01-02", v); err == nil {
			return &t
		}
		return nil
	}
	f.exact = parse(query.Get("DateSent"))
	if v := parse(query.Get("DateSent>")); v != nil {
		after := v.AddDate(0, 0, 1)
		f.after = &after
	}
	if v := parse(query.Get("DateSent>=")); v != nil {
		f.after = v
	}
	if v := parse(query.Get("DateSent<")); v != nil {
		f.before = v
	}
	if v := parse(query.Get("DateSent<=")); v != nil {
		before := v.AddDate(0, 0, 1)
		f.before = &before
	}
	return f
}

// matchesDate reports whether a message satisfies the DateSent filters.
func (f dateFilter) matches(t time.Time) bool {
	day := t.UTC().Truncate(24 * time.Hour)
	if f.exact != nil && !day.Equal(*f.exact) {
		return false
	}
	if f.after != nil && day.Before(*f.after) {
		return false
	}
	if f.before != nil && !day.Before(*f.before) {
		return false
	}
	return true
}

// listMessages handles GET /2010-04-01/Accounts/{AC}/Messages.json with
// To, From and DateSent filters plus PageSize/Page paging.
func (a *Adapter) listMessages(w http.ResponseWriter, r *http.Request) {
	accountSid := chi.URLParam(r, "AccountSid")
	query := r.URL.Query()

	filter := core.MessageFilter{Limit: 50}
	if to := strings.TrimSpace(query.Get("To")); to != "" {
		normalized := core.NormalizePhone(to)
		filter.To = &normalized
	}
	if from := strings.TrimSpace(query.Get("From")); from != "" {
		filter.From = &from
	}

	pageSize := 50
	if v, err := strconv.Atoi(query.Get("PageSize")); err == nil && v > 0 {
		pageSize = v
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	page := 0
	if v, err := strconv.Atoi(query.Get("Page")); err == nil && v > 0 {
		page = v
	}
	// Fetch up to the scan cap and paginate in memory so totals stay exact.
	filter.Limit = maxScanLimit

	msgs, _, err := a.service.Store().ListMessages(r.Context(), adapterkit.ProjectID(r), filter)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}

	dates := parseDateFilter(query)
	matched := make([]*core.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Provider != a.Name() {
			continue
		}
		if !dates.matches(m.CreatedAt) {
			continue
		}
		matched = append(matched, m)
	}

	total := len(matched)
	numPages := 0
	if total > 0 {
		numPages = (total + pageSize - 1) / pageSize
	}
	start := page * pageSize
	if start > total {
		start = total
	}
	end := start
	pageItems := []messagePayload{}
	if start < total {
		end = start + pageSize
		if end > total {
			end = total
		}
		pageItems = make([]messagePayload, 0, end-start)
		for _, m := range matched[start:end] {
			pageItems = append(pageItems, renderMessage(accountSid, m, 0))
		}
	}

	base := "/2010-04-01/Accounts/" + accountSid + "/Messages.json"
	pageURI := func(p int) string {
		v := url.Values{}
		if to := query.Get("To"); to != "" {
			v.Set("To", to)
		}
		if from := query.Get("From"); from != "" {
			v.Set("From", from)
		}
		if ds := query.Get("DateSent"); ds != "" {
			v.Set("DateSent", ds)
		}
		v.Set("PageSize", strconv.Itoa(pageSize))
		v.Set("Page", strconv.Itoa(p))
		return base + "?" + v.Encode()
	}
	var nextPageURI, prevPageURI *string
	if page+1 < numPages {
		u := pageURI(page + 1)
		nextPageURI = &u
	}
	if page > 0 {
		u := pageURI(page - 1)
		prevPageURI = &u
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listPayload{
		Messages:        pageItems,
		Total:           total,
		Page:            page,
		NumPages:        numPages,
		PageSize:        pageSize,
		Start:           start,
		End:             end,
		FirstPageURI:    pageURI(0),
		NextPageURI:     nextPageURI,
		PreviousPageURI: prevPageURI,
		URI:             pageURI(page),
	})
}

// fetchMessage handles GET /2010-04-01/Accounts/{AC}/Messages/{SM}.json.
func (a *Adapter) fetchMessage(w http.ResponseWriter, r *http.Request) {
	accountSid := chi.URLParam(r, "AccountSid")
	sid := chi.URLParam(r, "MessageSid")

	msgs, _, err := a.service.Store().ListMessages(r.Context(), adapterkit.ProjectID(r), core.MessageFilter{Limit: maxScanLimit})
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	for _, m := range msgs {
		if m.Provider == a.Name() && m.ProviderRef == sid {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(renderMessage(accountSid, m, 0))
			return
		}
	}

	a.WriteError(w, core.NewNotFound("The requested resource "+messageURI(accountSid, sid)+" was not found.", "sid"))
}
