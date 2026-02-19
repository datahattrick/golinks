package db

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func setupEditRequestTestDB(t *testing.T) (*DB, *models.User, *models.Link, func()) {
	t.Helper()
	db, baseCleanup := setupTestDB(t)

	ctx := context.Background()

	// Clean edit requests too
	db.Pool.Exec(ctx, "DELETE FROM link_edit_requests")

	user := &models.User{
		Sub:   "edit-req-user",
		Email: "editreq@example.com",
		Name:  "Edit Requester",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.Link{
		Keyword:   "editable-link",
		URL:       "https://example.com/old",
		Scope:     models.ScopeGlobal,
		CreatedBy: &user.ID,
	}
	if err := db.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	cleanup := func() {
		db.Pool.Exec(ctx, "DELETE FROM link_edit_requests")
		baseCleanup()
	}

	return db, user, link, cleanup
}

func TestCreateEditRequest(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID:      link.ID,
		UserID:      user.ID,
		URL:         "https://example.com/new",
		Description: "Updated description",
		Reason:      "URL was outdated",
	}

	err := db.CreateEditRequest(ctx, req)
	if err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	if req.ID == uuid.Nil {
		t.Error("CreateEditRequest() did not set ID")
	}
	if req.Status != models.StatusPending {
		t.Errorf("CreateEditRequest() status = %q, want %q", req.Status, models.StatusPending)
	}
	if req.CreatedAt.IsZero() {
		t.Error("CreateEditRequest() did not set CreatedAt")
	}
}

func TestCreateEditRequest_DuplicatePerLink(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req1 := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/new1",
		Reason: "First request",
	}
	if err := db.CreateEditRequest(ctx, req1); err != nil {
		t.Fatalf("CreateEditRequest() first error = %v", err)
	}

	req2 := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/new2",
		Reason: "Second request",
	}
	err := db.CreateEditRequest(ctx, req2)
	if err != ErrDuplicateEditRequest {
		t.Errorf("CreateEditRequest() error = %v, want ErrDuplicateEditRequest", err)
	}
}

func TestCreateEditRequest_PendingLimit(t *testing.T) {
	db, user, _, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create 5 different links to attach requests to
	for i := range 5 {
		l := &models.Link{
			Keyword:   "limit-link-" + string(rune('a'+i)),
			URL:       "https://example.com/limit",
			Scope:     models.ScopeGlobal,
			CreatedBy: &user.ID,
		}
		if err := db.CreateLink(ctx, l); err != nil {
			t.Fatalf("CreateLink(%d) error = %v", i, err)
		}

		req := &models.LinkEditRequest{
			LinkID: l.ID,
			UserID: user.ID,
			URL:    "https://example.com/updated",
			Reason: "Request",
		}
		if err := db.CreateEditRequest(ctx, req); err != nil {
			t.Fatalf("CreateEditRequest(%d) error = %v", i, err)
		}
	}

	// 6th link + request should hit the limit
	l6 := &models.Link{
		Keyword:   "limit-link-f",
		URL:       "https://example.com/limit6",
		Scope:     models.ScopeGlobal,
		CreatedBy: &user.ID,
	}
	if err := db.CreateLink(ctx, l6); err != nil {
		t.Fatalf("CreateLink(6) error = %v", err)
	}

	req6 := &models.LinkEditRequest{
		LinkID: l6.ID,
		UserID: user.ID,
		URL:    "https://example.com/updated6",
		Reason: "Over limit",
	}
	err := db.CreateEditRequest(ctx, req6)
	if err != ErrPendingRequestLimit {
		t.Errorf("CreateEditRequest() error = %v, want ErrPendingRequestLimit", err)
	}
}

func TestGetEditRequestByID(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID:      link.ID,
		UserID:      user.ID,
		URL:         "https://example.com/fetched",
		Description: "Fetch me",
		Reason:      "Testing retrieval",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	fetched, err := db.GetEditRequestByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetEditRequestByID() error = %v", err)
	}

	if fetched.URL != "https://example.com/fetched" {
		t.Errorf("GetEditRequestByID() URL = %q, want %q", fetched.URL, "https://example.com/fetched")
	}
	if fetched.Keyword != "editable-link" {
		t.Errorf("GetEditRequestByID() Keyword = %q, want %q", fetched.Keyword, "editable-link")
	}
	if fetched.AuthorName != "Edit Requester" {
		t.Errorf("GetEditRequestByID() AuthorName = %q, want %q", fetched.AuthorName, "Edit Requester")
	}
}

func TestGetEditRequestByID_NotFound(t *testing.T) {
	db, _, _, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	_, err := db.GetEditRequestByID(context.Background(), uuid.New())
	if err != ErrEditRequestNotFound {
		t.Errorf("GetEditRequestByID() error = %v, want ErrEditRequestNotFound", err)
	}
}

func TestGetPendingEditRequests_GlobalMod(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/pending",
		Reason: "Pending review",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	mod := &models.User{Role: models.RoleGlobalMod}
	requests, err := db.GetPendingEditRequests(ctx, mod)
	if err != nil {
		t.Fatalf("GetPendingEditRequests() error = %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("GetPendingEditRequests() returned %d, want 1", len(requests))
	}
}

func TestGetPendingEditRequests_RegularUser(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/pending",
		Reason: "Pending review",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	regularUser := &models.User{Role: models.RoleUser}
	requests, err := db.GetPendingEditRequests(ctx, regularUser)
	if err != nil {
		t.Fatalf("GetPendingEditRequests() error = %v", err)
	}

	if len(requests) != 0 {
		t.Errorf("GetPendingEditRequests() returned %d for regular user, want 0", len(requests))
	}
}

func TestApproveEditRequest(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID:      link.ID,
		UserID:      user.ID,
		URL:         "https://example.com/approved-url",
		Description: "Approved description",
		Reason:      "Fix typo",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	reviewer := &models.User{
		Sub:   "edit-reviewer",
		Email: "reviewer@example.com",
		Name:  "Reviewer",
	}
	if err := db.UpsertUser(ctx, reviewer); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	err := db.ApproveEditRequest(ctx, req.ID, reviewer.ID)
	if err != nil {
		t.Fatalf("ApproveEditRequest() error = %v", err)
	}

	// Verify link was updated
	updatedLink, err := db.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID() error = %v", err)
	}
	if updatedLink.URL != "https://example.com/approved-url" {
		t.Errorf("ApproveEditRequest() link URL = %q, want %q", updatedLink.URL, "https://example.com/approved-url")
	}
	if updatedLink.Description != "Approved description" {
		t.Errorf("ApproveEditRequest() link Description = %q, want %q", updatedLink.Description, "Approved description")
	}
	if updatedLink.HealthStatus != models.HealthUnknown {
		t.Errorf("ApproveEditRequest() health_status = %q, want %q", updatedLink.HealthStatus, models.HealthUnknown)
	}

	// Verify the request is now approved
	approved, err := db.GetEditRequestByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetEditRequestByID() error = %v", err)
	}
	if approved.Status != models.StatusApproved {
		t.Errorf("ApproveEditRequest() request status = %q, want %q", approved.Status, models.StatusApproved)
	}
	if approved.ReviewedBy == nil || *approved.ReviewedBy != reviewer.ID {
		t.Error("ApproveEditRequest() did not set ReviewedBy")
	}
}

func TestApproveEditRequest_NotFound(t *testing.T) {
	db, _, _, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	err := db.ApproveEditRequest(context.Background(), uuid.New(), uuid.New())
	if err != ErrEditRequestNotFound {
		t.Errorf("ApproveEditRequest() error = %v, want ErrEditRequestNotFound", err)
	}
}

func TestRejectEditRequest(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/rejected",
		Reason: "Bad edit",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	reviewer := &models.User{
		Sub:   "reject-reviewer",
		Email: "reject-reviewer@example.com",
		Name:  "Reject Reviewer",
	}
	if err := db.UpsertUser(ctx, reviewer); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	err := db.RejectEditRequest(ctx, req.ID, reviewer.ID)
	if err != nil {
		t.Fatalf("RejectEditRequest() error = %v", err)
	}

	rejected, err := db.GetEditRequestByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetEditRequestByID() error = %v", err)
	}
	if rejected.Status != models.StatusRejected {
		t.Errorf("RejectEditRequest() status = %q, want %q", rejected.Status, models.StatusRejected)
	}
}

func TestRejectEditRequest_NotFound(t *testing.T) {
	db, _, _, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	err := db.RejectEditRequest(context.Background(), uuid.New(), uuid.New())
	if err != ErrEditRequestNotFound {
		t.Errorf("RejectEditRequest() error = %v, want ErrEditRequestNotFound", err)
	}
}

func TestGetLinkIDsWithPendingEdits(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a second link without an edit request
	link2 := &models.Link{
		Keyword:   "no-edit-link",
		URL:       "https://example.com/noedit",
		Scope:     models.ScopeGlobal,
		CreatedBy: &user.ID,
	}
	if err := db.CreateLink(ctx, link2); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	// Add an edit request only for the first link
	req := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/edited",
		Reason: "Testing pending edits map",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	result, err := db.GetLinkIDsWithPendingEdits(ctx, []uuid.UUID{link.ID, link2.ID})
	if err != nil {
		t.Fatalf("GetLinkIDsWithPendingEdits() error = %v", err)
	}

	if !result[link.ID.String()] {
		t.Error("GetLinkIDsWithPendingEdits() missing link with pending edit")
	}
	if result[link2.ID.String()] {
		t.Error("GetLinkIDsWithPendingEdits() incorrectly included link without pending edit")
	}
}

func TestGetLinkIDsWithPendingEdits_EmptyInput(t *testing.T) {
	db, _, _, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	result, err := db.GetLinkIDsWithPendingEdits(context.Background(), []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetLinkIDsWithPendingEdits() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("GetLinkIDsWithPendingEdits() returned %d results for empty input, want 0", len(result))
	}
}

func TestGetPendingEditRequestForLink(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	req := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/check",
		Reason: "Check pending",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	found, err := db.GetPendingEditRequestForLink(ctx, link.ID, user.ID)
	if err != nil {
		t.Fatalf("GetPendingEditRequestForLink() error = %v", err)
	}
	if found.ID != req.ID {
		t.Errorf("GetPendingEditRequestForLink() ID = %v, want %v", found.ID, req.ID)
	}
}

func TestGetPendingEditRequestForLink_NotFound(t *testing.T) {
	db, _, _, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	_, err := db.GetPendingEditRequestForLink(context.Background(), uuid.New(), uuid.New())
	if err != ErrEditRequestNotFound {
		t.Errorf("GetPendingEditRequestForLink() error = %v, want ErrEditRequestNotFound", err)
	}
}

func TestCreateEditRequest_AllowedAfterRejection(t *testing.T) {
	db, user, link, cleanup := setupEditRequestTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create and reject a request
	req := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/first",
		Reason: "First attempt",
	}
	if err := db.CreateEditRequest(ctx, req); err != nil {
		t.Fatalf("CreateEditRequest() error = %v", err)
	}

	reviewer := &models.User{
		Sub:   "rejection-reviewer",
		Email: "rej-reviewer@example.com",
		Name:  "Rejection Reviewer",
	}
	if err := db.UpsertUser(ctx, reviewer); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}
	if err := db.RejectEditRequest(ctx, req.ID, reviewer.ID); err != nil {
		t.Fatalf("RejectEditRequest() error = %v", err)
	}

	// Should be able to create a new request for the same link now
	req2 := &models.LinkEditRequest{
		LinkID: link.ID,
		UserID: user.ID,
		URL:    "https://example.com/second",
		Reason: "Second attempt after rejection",
	}
	err := db.CreateEditRequest(ctx, req2)
	if err != nil {
		t.Errorf("CreateEditRequest() after rejection error = %v, want nil", err)
	}
}
