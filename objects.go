package pageweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ObjectsService operates on typed business records: the values held under an ObjectTypesService
// type. Reads need objects:read (plus objects:read-sensitive to decrypt sensitive fields); writes
// need objects:write; relationships and document links need relationships:manage.
type ObjectsService struct {
	client *Client
}

// ListObjectsParams filters an object listing.
type ListObjectsParams struct {
	ObjectTypeKey  string
	ObjectTypeID   string
	Status         string // "active" | "archived"
	LifecycleState string
	OwnerUserID    string
	Number         string
	Cursor         string
	Limit          int
}

// List returns objects. Rows never carry field data — Get one for that.
func (s *ObjectsService) List(ctx context.Context, params ListObjectsParams) (Document, error) {
	q := url.Values{}
	setQuery(q, "objectTypeKey", params.ObjectTypeKey)
	setQuery(q, "objectTypeId", params.ObjectTypeID)
	setQuery(q, "status", params.Status)
	setQuery(q, "lifecycleState", params.LifecycleState)
	setQuery(q, "ownerUserId", params.OwnerUserID)
	setQuery(q, "number", params.Number)
	setQuery(q, "cursor", params.Cursor)
	setQueryInt(q, "limit", params.Limit)
	return s.client.do(ctx, http.MethodGet, "/v1/objects", q, nil, nil, false)
}

// GetOptions controls Get.
type GetOptions struct {
	// Version fetches a specific past version's value instead of the current one.
	Version int
	// IncludeSensitive decrypts sensitive fields (requires the objects:read-sensitive scope; a key
	// without it gets a 403, never a silently redacted response).
	IncludeSensitive bool
}

// Get fetches one object's current (or a specific Version's) value.
func (s *ObjectsService) Get(ctx context.Context, id string, opts GetOptions) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQueryInt(q, "version", opts.Version)
	setQueryBool(q, "includeSensitive", opts.IncludeSensitive)
	return s.client.do(ctx, http.MethodGet, "/v1/objects/"+url.PathEscape(id), q, nil, nil, false)
}

// Create creates an object. body must set exactly one of "objectTypeKey"/"objectTypeId", and a
// "data" object. Pass an idempotencyKey (or "") to make a retried create return the original record
// instead of creating a duplicate; the same key with a different body is a 409.
func (s *ObjectsService) Create(ctx context.Context, body map[string]any, idempotencyKey string) (Document, error) {
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	typeKey, _ := body["objectTypeKey"].(string)
	typeID, _ := body["objectTypeId"].(string)
	if err := requireOneOf(typeKey, "objectTypeKey", typeID, "objectTypeId"); err != nil {
		return nil, err
	}
	data, _ := body["data"].(map[string]any)
	if err := requireObjectBody(data, "body.data"); err != nil {
		return nil, err
	}
	var headers map[string]string
	if idempotencyKey != "" {
		headers = map[string]string{"idempotency-key": idempotencyKey}
	}
	return s.client.do(ctx, http.MethodPost, "/v1/objects", nil, body, headers, false)
}

// Replace replaces an object's whole value (never merged). body must set "data" and
// "expectedVersion" (int) — an optimistic concurrency check the API enforces with a 409 on
// mismatch, so a lost update never overwrites someone else's change silently.
func (s *ObjectsService) Replace(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	data, _ := body["data"].(map[string]any)
	if err := requireObjectBody(data, "body.data"); err != nil {
		return nil, err
	}
	expectedVersion, _ := body["expectedVersion"].(int)
	if v, ok := body["expectedVersion"].(float64); ok {
		expectedVersion = int(v)
	}
	if err := requirePositiveInt(expectedVersion, "body.expectedVersion"); err != nil {
		return nil, err
	}
	headers := map[string]string{"if-match": fmt.Sprintf("%d", expectedVersion)}
	return s.client.do(ctx, http.MethodPut, "/v1/objects/"+url.PathEscape(id), nil, body, headers, false)
}

// Versions returns version history (metadata only, never values — read a version's value via Get
// with Version set).
func (s *ObjectsService) Versions(ctx context.Context, id, cursor string, limit int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, "/v1/objects/"+url.PathEscape(id)+"/versions", q, nil, nil, false)
}

// Archive archives an object (reversible via Restore). body must set "reason".
func (s *ObjectsService) Archive(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	reason, _ := body["reason"].(string)
	if err := requireString(reason, "body.reason"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/objects/"+url.PathEscape(id)+"/archive", nil, body, nil, false)
}

// Restore restores an archived object. No new version is created — status only.
func (s *ObjectsService) Restore(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/objects/"+url.PathEscape(id)+"/restore", nil, nil, nil, false)
}

// Relationships lists relationship edges to/from this object, in both directions. Pass
// includeEnded true to include ended ones.
func (s *ObjectsService) Relationships(ctx context.Context, id string, includeEnded bool) ([]Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQueryBool(q, "includeEnded", includeEnded)
	return s.client.doList(ctx, http.MethodGet, "/v1/objects/"+url.PathEscape(id)+"/relationships", q, nil, nil)
}

// AddRelationship creates a relationship from this object (the source) to
// body["targetObjectId"]. Provide exactly one of body["relationshipTypeKey"]/
// body["relationshipTypeId"]. Refused (with a reason) when the endpoint types aren't allowed,
// cardinality is already satisfied, either record is archived, or the target is in a different
// account. unchanged: true on the result means an identical live edge already existed.
func (s *ObjectsService) AddRelationship(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	typeKey, _ := body["relationshipTypeKey"].(string)
	typeID, _ := body["relationshipTypeId"].(string)
	if err := requireOneOf(typeKey, "relationshipTypeKey", typeID, "relationshipTypeId"); err != nil {
		return nil, err
	}
	target, _ := body["targetObjectId"].(string)
	if err := requireString(target, "body.targetObjectId"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/objects/"+url.PathEscape(id)+"/relationships", nil, body, nil, false)
}

// EndRelationship ends a relationship. The row stays (with validTo set); nothing is deleted.
func (s *ObjectsService) EndRelationship(ctx context.Context, id, relationshipID string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if _, err := requireID(relationshipID, "relationshipId"); err != nil {
		return nil, err
	}
	path := "/v1/objects/" + url.PathEscape(id) + "/relationships/" + url.PathEscape(relationshipID) + "/end"
	return s.client.do(ctx, http.MethodPost, path, nil, body, nil, false)
}

// Documents lists documents filed against this object.
func (s *ObjectsService) Documents(ctx context.Context, id string) ([]Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.doList(ctx, http.MethodGet, "/v1/objects/"+url.PathEscape(id)+"/documents", nil, nil, nil)
}

// LinkDocument files a document against this object. body must set "documentId"; optionally
// "role" (default "primary"). Idempotent per (document, object, role).
func (s *ObjectsService) LinkDocument(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	documentID, _ := body["documentId"].(string)
	if err := requireString(documentID, "body.documentId"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/objects/"+url.PathEscape(id)+"/documents", nil, body, nil, false)
}

// UnlinkDocument unfiles a document link. role filters which role to unlink (pass "" for the
// default). Idempotent: unlinking an absent link succeeds with removed: false.
func (s *ObjectsService) UnlinkDocument(ctx context.Context, id, documentID, role string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if _, err := requireID(documentID, "documentId"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQuery(q, "role", role)
	path := "/v1/objects/" + url.PathEscape(id) + "/documents/" + url.PathEscape(documentID)
	return s.client.do(ctx, http.MethodDelete, path, q, nil, nil, false)
}
