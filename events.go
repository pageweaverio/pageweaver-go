package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// EventsService reads the append-only domain-event ledger: what happened, in order, for
// correlation and replay. Entries are filtered to what the calling key's scopes can see (a key
// without objects:read sees nothing about object-model events); hidden entries are silently
// dropped, not an error. Requires only the baseline read scope every key has.
type EventsService struct {
	client *Client
}

// ListEventsParams filters an events page. After is the resume point (exclusive) — always resume
// from the returned nextCursor, even if it doesn't equal the last event you saw (some may have
// been scope-trimmed).
type ListEventsParams struct {
	After         string
	Limit         int
	Type          string
	SubjectType   string
	SubjectID     string
	CorrelationID string
}

// List returns one page of events.
func (s *EventsService) List(ctx context.Context, params ListEventsParams) (Document, error) {
	q := url.Values{}
	setQuery(q, "after", params.After)
	setQueryInt(q, "limit", params.Limit)
	setQuery(q, "type", params.Type)
	setQuery(q, "subjectType", params.SubjectType)
	setQuery(q, "subjectId", params.SubjectID)
	setQuery(q, "correlationId", params.CorrelationID)
	return s.client.do(ctx, http.MethodGet, "/v1/events", q, nil, nil, false)
}

// ListAll iterates every visible event forward from params.After (or the beginning), transparently
// following nextCursor, invoking fn for each event in order. Stops once a page comes back with no
// events (you have caught up) or fn returns an error. Returns the last cursor observed, so the
// caller can resume later by passing it back as After.
func (s *EventsService) ListAll(ctx context.Context, params ListEventsParams, fn func(Document) error) (string, error) {
	after := params.After
	for {
		page, err := s.List(ctx, ListEventsParams{
			After: after, Limit: params.Limit, Type: params.Type,
			SubjectType: params.SubjectType, SubjectID: params.SubjectID, CorrelationID: params.CorrelationID,
		})
		if err != nil {
			return after, err
		}
		events, _ := page["events"].([]any)
		if len(events) == 0 {
			return after, nil
		}
		for _, e := range events {
			if m, ok := e.(map[string]any); ok {
				if err := fn(Document(m)); err != nil {
					return after, err
				}
			}
		}
		next, _ := page["nextCursor"].(string)
		if next == "" {
			return after, nil
		}
		after = next
	}
}
