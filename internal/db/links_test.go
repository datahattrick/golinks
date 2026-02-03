package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func skipIfNoTestDB(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" && os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}
}

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()
	skipIfNoTestDB(t)

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		connString = "postgres://golinks:golinks@localhost:5432/golinks_test?sslmode=disable"
	}

	ctx := context.Background()
	database, err := New(ctx, connString)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := database.RunMigrations(connString); err != nil {
		database.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		// Clean up in order
		database.Pool.Exec(ctx, "DELETE FROM user_links")
		database.Pool.Exec(ctx, "DELETE FROM links")
		database.Pool.Exec(ctx, "DELETE FROM users")
		database.Pool.Exec(ctx, "DELETE FROM organizations")
		database.Close()
	}

	// Clean before test
	database.Pool.Exec(ctx, "DELETE FROM user_links")
	database.Pool.Exec(ctx, "DELETE FROM links")
	database.Pool.Exec(ctx, "DELETE FROM users")
	database.Pool.Exec(ctx, "DELETE FROM organizations")

	return database, cleanup
}

func TestCreateLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "test-creator",
		Email: "creator@example.com",
		Name:  "Test Creator",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.Link{
		Keyword:     "test-link",
		URL:         "https://example.com",
		Description: "Test description",
		Scope:       models.ScopeGlobal,
		CreatedBy:   &user.ID,
	}

	err := db.CreateLink(ctx, link)
	if err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	if link.ID == uuid.Nil {
		t.Error("CreateLink() did not set ID")
	}
	if link.Status != models.StatusApproved {
		t.Errorf("CreateLink() status = %q, want %q", link.Status, models.StatusApproved)
	}
}

func TestCreateLink_DuplicateKeyword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	link1 := &models.Link{
		Keyword: "duplicate-test",
		URL:     "https://example1.com",
		Scope:   models.ScopeGlobal,
	}
	if err := db.CreateLink(ctx, link1); err != nil {
		t.Fatalf("CreateLink() first link error = %v", err)
	}

	link2 := &models.Link{
		Keyword: "duplicate-test",
		URL:     "https://example2.com",
		Scope:   models.ScopeGlobal,
	}
	err := db.CreateLink(ctx, link2)
	if err != ErrDuplicateKeyword {
		t.Errorf("CreateLink() error = %v, want ErrDuplicateKeyword", err)
	}
}

func TestSubmitLinkForApproval(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "test-submitter",
		Email: "submitter@example.com",
		Name:  "Test Submitter",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.Link{
		Keyword:     "pending-link",
		URL:         "https://example.com",
		Description: "Pending link",
		Scope:       models.ScopeGlobal,
		SubmittedBy: &user.ID,
	}

	err := db.SubmitLinkForApproval(ctx, link)
	if err != nil {
		t.Fatalf("SubmitLinkForApproval() error = %v", err)
	}

	if link.Status != models.StatusPending {
		t.Errorf("SubmitLinkForApproval() status = %q, want %q", link.Status, models.StatusPending)
	}
}

func TestApproveLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	submitter := &models.User{
		Sub:   "test-submitter-approve",
		Email: "submitter-approve@example.com",
		Name:  "Test Submitter",
	}
	if err := db.UpsertUser(ctx, submitter); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	reviewer := &models.User{
		Sub:   "test-reviewer-approve",
		Email: "reviewer-approve@example.com",
		Name:  "Test Reviewer",
	}
	if err := db.UpsertUser(ctx, reviewer); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.Link{
		Keyword:     "approve-test",
		URL:         "https://example.com",
		Scope:       models.ScopeGlobal,
		SubmittedBy: &submitter.ID,
	}
	if err := db.SubmitLinkForApproval(ctx, link); err != nil {
		t.Fatalf("SubmitLinkForApproval() error = %v", err)
	}

	err := db.ApproveLink(ctx, link.ID, reviewer.ID)
	if err != nil {
		t.Fatalf("ApproveLink() error = %v", err)
	}

	// Verify the link is now approved
	approved, err := db.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID() error = %v", err)
	}
	if approved.Status != models.StatusApproved {
		t.Errorf("ApproveLink() status = %q, want %q", approved.Status, models.StatusApproved)
	}
	if approved.ReviewedBy == nil || *approved.ReviewedBy != reviewer.ID {
		t.Error("ApproveLink() did not set ReviewedBy")
	}
}

func TestRejectLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	submitter := &models.User{
		Sub:   "test-submitter-reject",
		Email: "submitter-reject@example.com",
		Name:  "Test Submitter",
	}
	if err := db.UpsertUser(ctx, submitter); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	reviewer := &models.User{
		Sub:   "test-reviewer-reject",
		Email: "reviewer-reject@example.com",
		Name:  "Test Reviewer",
	}
	if err := db.UpsertUser(ctx, reviewer); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.Link{
		Keyword:     "reject-test",
		URL:         "https://example.com",
		Scope:       models.ScopeGlobal,
		SubmittedBy: &submitter.ID,
	}
	if err := db.SubmitLinkForApproval(ctx, link); err != nil {
		t.Fatalf("SubmitLinkForApproval() error = %v", err)
	}

	err := db.RejectLink(ctx, link.ID, reviewer.ID)
	if err != nil {
		t.Fatalf("RejectLink() error = %v", err)
	}

	rejected, err := db.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID() error = %v", err)
	}
	if rejected.Status != models.StatusRejected {
		t.Errorf("RejectLink() status = %q, want %q", rejected.Status, models.StatusRejected)
	}
}

func TestGetApprovedGlobalLinkByKeyword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create an approved global link
	link := &models.Link{
		Keyword: "global-link",
		URL:     "https://global.example.com",
		Scope:   models.ScopeGlobal,
		Status:  models.StatusApproved,
	}
	if err := db.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	// Should find the link
	found, err := db.GetApprovedGlobalLinkByKeyword(ctx, "global-link")
	if err != nil {
		t.Fatalf("GetApprovedGlobalLinkByKeyword() error = %v", err)
	}
	if found.URL != "https://global.example.com" {
		t.Errorf("GetApprovedGlobalLinkByKeyword() URL = %q, want %q", found.URL, "https://global.example.com")
	}

	// Should not find non-existent link
	_, err = db.GetApprovedGlobalLinkByKeyword(ctx, "non-existent")
	if err != ErrLinkNotFound {
		t.Errorf("GetApprovedGlobalLinkByKeyword() error = %v, want ErrLinkNotFound", err)
	}
}

func TestGetPendingGlobalLinks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "test-pending-submitter",
		Email: "pending-submitter@example.com",
		Name:  "Test Submitter",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	// Create some pending links
	for i := 1; i <= 3; i++ {
		link := &models.Link{
			Keyword:     "pending-" + string(rune('0'+i)),
			URL:         "https://example.com",
			Scope:       models.ScopeGlobal,
			SubmittedBy: &user.ID,
		}
		if err := db.SubmitLinkForApproval(ctx, link); err != nil {
			t.Fatalf("SubmitLinkForApproval() error = %v", err)
		}
	}

	pending, err := db.GetPendingGlobalLinks(ctx)
	if err != nil {
		t.Fatalf("GetPendingGlobalLinks() error = %v", err)
	}
	if len(pending) != 3 {
		t.Errorf("GetPendingGlobalLinks() returned %d links, want 3", len(pending))
	}
}

func TestSearchApprovedLinks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create some approved links
	links := []struct {
		keyword string
		url     string
	}{
		{"google", "https://google.com"},
		{"github", "https://github.com"},
		{"golang", "https://go.dev"},
	}
	for _, l := range links {
		link := &models.Link{
			Keyword: l.keyword,
			URL:     l.url,
			Scope:   models.ScopeGlobal,
		}
		if err := db.CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink() error = %v", err)
		}
	}

	// Search for "go"
	results, err := db.SearchApprovedLinks(ctx, "go", nil, 10)
	if err != nil {
		t.Fatalf("SearchApprovedLinks() error = %v", err)
	}
	if len(results) != 2 { // google and golang
		t.Errorf("SearchApprovedLinks('go') returned %d results, want 2", len(results))
	}

	// Search with empty query returns all
	all, err := db.SearchApprovedLinks(ctx, "", nil, 10)
	if err != nil {
		t.Fatalf("SearchApprovedLinks('') error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("SearchApprovedLinks('') returned %d results, want 3", len(all))
	}
}

func TestDeleteLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	link := &models.Link{
		Keyword: "delete-test",
		URL:     "https://example.com",
		Scope:   models.ScopeGlobal,
	}
	if err := db.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	err := db.DeleteLink(ctx, link.ID)
	if err != nil {
		t.Fatalf("DeleteLink() error = %v", err)
	}

	// Verify deleted
	_, err = db.GetLinkByID(ctx, link.ID)
	if err != ErrLinkNotFound {
		t.Errorf("GetLinkByID() after delete error = %v, want ErrLinkNotFound", err)
	}
}

func TestIncrementClickCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	link := &models.Link{
		Keyword: "click-test",
		URL:     "https://example.com",
		Scope:   models.ScopeGlobal,
	}
	if err := db.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	// Increment multiple times
	for range 5 {
		if err := db.IncrementClickCount(ctx, link.ID); err != nil {
			t.Fatalf("IncrementClickCount() error = %v", err)
		}
	}

	updated, err := db.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID() error = %v", err)
	}
	if updated.ClickCount != 5 {
		t.Errorf("IncrementClickCount() click_count = %d, want 5", updated.ClickCount)
	}
}

func TestGetTopUsedLinksForUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := uuid.New()

	// Create some global links with varying click counts
	linksData := []struct {
		keyword string
		clicks  int
	}{
		{"top-link-1", 100},
		{"top-link-2", 50},
		{"top-link-3", 75},
		{"top-link-4", 25},
		{"top-link-5", 10},
		{"top-link-6", 5},
	}

	for _, l := range linksData {
		link := &models.Link{
			Keyword: l.keyword,
			URL:     "https://example.com/" + l.keyword,
			Scope:   models.ScopeGlobal,
		}
		if err := db.CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink() error = %v", err)
		}
		// Set click count
		for i := 0; i < l.clicks; i++ {
			if err := db.IncrementClickCount(ctx, link.ID); err != nil {
				t.Fatalf("IncrementClickCount() error = %v", err)
			}
		}
	}

	// Get top 5 used links
	topLinks, err := db.GetTopUsedLinksForUser(ctx, userID, nil, 5)
	if err != nil {
		t.Fatalf("GetTopUsedLinksForUser() error = %v", err)
	}

	if len(topLinks) != 5 {
		t.Errorf("GetTopUsedLinksForUser() returned %d links, want 5", len(topLinks))
	}

	// Verify ordering by click count (descending)
	if len(topLinks) > 0 && topLinks[0].Keyword != "top-link-1" {
		t.Errorf("GetTopUsedLinksForUser() first link = %q, want %q", topLinks[0].Keyword, "top-link-1")
	}
	if len(topLinks) > 1 && topLinks[1].Keyword != "top-link-3" {
		t.Errorf("GetTopUsedLinksForUser() second link = %q, want %q", topLinks[1].Keyword, "top-link-3")
	}
}

func TestGetTopUsedLinksForUser_WithOrg(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create organization
	org := &models.Organization{Name: "Test Org", Slug: "test-org"}
	if err := db.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	userID := uuid.New()

	// Create a global link
	globalLink := &models.Link{
		Keyword: "global-top",
		URL:     "https://global.example.com",
		Scope:   models.ScopeGlobal,
	}
	if err := db.CreateLink(ctx, globalLink); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}
	for range 50 {
		db.IncrementClickCount(ctx, globalLink.ID)
	}

	// Create an org link
	orgLink := &models.Link{
		Keyword:        "org-top",
		URL:            "https://org.example.com",
		Scope:          models.ScopeOrg,
		OrganizationID: &org.ID,
	}
	if err := db.CreateLink(ctx, orgLink); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}
	for range 100 {
		db.IncrementClickCount(ctx, orgLink.ID)
	}

	// Get top links with org ID
	topLinks, err := db.GetTopUsedLinksForUser(ctx, userID, &org.ID, 5)
	if err != nil {
		t.Fatalf("GetTopUsedLinksForUser() error = %v", err)
	}

	if len(topLinks) != 2 {
		t.Errorf("GetTopUsedLinksForUser() returned %d links, want 2", len(topLinks))
	}

	// Org link should be first (higher clicks)
	if len(topLinks) > 0 && topLinks[0].Keyword != "org-top" {
		t.Errorf("GetTopUsedLinksForUser() first link = %q, want %q", topLinks[0].Keyword, "org-top")
	}
}

func TestGetNewestApprovedLinks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create links - they will be created in order
	keywords := []string{"newest-1", "newest-2", "newest-3", "newest-4", "newest-5", "newest-6"}
	for _, kw := range keywords {
		link := &models.Link{
			Keyword: kw,
			URL:     "https://example.com/" + kw,
			Scope:   models.ScopeGlobal,
		}
		if err := db.CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink() error = %v", err)
		}
	}

	// Get newest 5 links
	newestLinks, err := db.GetNewestApprovedLinks(ctx, nil, 5)
	if err != nil {
		t.Fatalf("GetNewestApprovedLinks() error = %v", err)
	}

	if len(newestLinks) != 5 {
		t.Errorf("GetNewestApprovedLinks() returned %d links, want 5", len(newestLinks))
	}

	// Verify ordering (newest first)
	if len(newestLinks) > 0 && newestLinks[0].Keyword != "newest-6" {
		t.Errorf("GetNewestApprovedLinks() first link = %q, want %q", newestLinks[0].Keyword, "newest-6")
	}
}

func TestGetNewestApprovedLinks_WithOrg(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create organization
	org := &models.Organization{Name: "Test Org", Slug: "test-org-newest"}
	if err := db.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	// Create a global link
	globalLink := &models.Link{
		Keyword: "global-newest",
		URL:     "https://global.example.com",
		Scope:   models.ScopeGlobal,
	}
	if err := db.CreateLink(ctx, globalLink); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	// Create an org link
	orgLink := &models.Link{
		Keyword:        "org-newest",
		URL:            "https://org.example.com",
		Scope:          models.ScopeOrg,
		OrganizationID: &org.ID,
	}
	if err := db.CreateLink(ctx, orgLink); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	// Get newest links with org ID - should include both
	newestLinks, err := db.GetNewestApprovedLinks(ctx, &org.ID, 5)
	if err != nil {
		t.Fatalf("GetNewestApprovedLinks() error = %v", err)
	}

	if len(newestLinks) != 2 {
		t.Errorf("GetNewestApprovedLinks() returned %d links, want 2", len(newestLinks))
	}

	// Org link is newer
	if len(newestLinks) > 0 && newestLinks[0].Keyword != "org-newest" {
		t.Errorf("GetNewestApprovedLinks() first link = %q, want %q", newestLinks[0].Keyword, "org-newest")
	}
}

func TestGetRandomApprovedLinks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create some links
	for i := 1; i <= 10; i++ {
		link := &models.Link{
			Keyword: "random-" + string(rune('0'+i)),
			URL:     "https://example.com/random",
			Scope:   models.ScopeGlobal,
		}
		if err := db.CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink() error = %v", err)
		}
	}

	// Get 5 random links
	randomLinks, err := db.GetRandomApprovedLinks(ctx, nil, 5)
	if err != nil {
		t.Fatalf("GetRandomApprovedLinks() error = %v", err)
	}

	if len(randomLinks) != 5 {
		t.Errorf("GetRandomApprovedLinks() returned %d links, want 5", len(randomLinks))
	}

	// Verify all links are approved
	for _, link := range randomLinks {
		if link.Status != models.StatusApproved {
			t.Errorf("GetRandomApprovedLinks() returned non-approved link: %s", link.Keyword)
		}
	}
}

func TestGetRandomApprovedLinks_ExcludesPending(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user first (for the foreign key constraint)
	user := &models.User{
		Sub:   "test-pending-user",
		Email: "pending@example.com",
		Name:  "Test User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	// Create an approved link
	approvedLink := &models.Link{
		Keyword: "approved-random",
		URL:     "https://approved.example.com",
		Scope:   models.ScopeGlobal,
	}
	if err := db.CreateLink(ctx, approvedLink); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	// Create a pending link
	pendingLink := &models.Link{
		Keyword:     "pending-random",
		URL:         "https://pending.example.com",
		Scope:       models.ScopeGlobal,
		SubmittedBy: &user.ID,
	}
	if err := db.SubmitLinkForApproval(ctx, pendingLink); err != nil {
		t.Fatalf("SubmitLinkForApproval() error = %v", err)
	}

	// Get random links - should only return approved
	randomLinks, err := db.GetRandomApprovedLinks(ctx, nil, 10)
	if err != nil {
		t.Fatalf("GetRandomApprovedLinks() error = %v", err)
	}

	if len(randomLinks) != 1 {
		t.Errorf("GetRandomApprovedLinks() returned %d links, want 1", len(randomLinks))
	}

	if len(randomLinks) > 0 && randomLinks[0].Keyword != "approved-random" {
		t.Errorf("GetRandomApprovedLinks() returned %q, want %q", randomLinks[0].Keyword, "approved-random")
	}
}

func TestGetRandomApprovedLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create some links
	for i := 1; i <= 5; i++ {
		link := &models.Link{
			Keyword: "single-random-" + string(rune('0'+i)),
			URL:     "https://example.com/single",
			Scope:   models.ScopeGlobal,
		}
		if err := db.CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink() error = %v", err)
		}
	}

	// Get a single random link
	randomLink, err := db.GetRandomApprovedLink(ctx, nil)
	if err != nil {
		t.Fatalf("GetRandomApprovedLink() error = %v", err)
	}

	if randomLink == nil {
		t.Error("GetRandomApprovedLink() returned nil")
	}

	if randomLink.Status != models.StatusApproved {
		t.Errorf("GetRandomApprovedLink() returned non-approved link")
	}
}

func TestGetRandomApprovedLink_NoLinks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get random link when none exist
	_, err := db.GetRandomApprovedLink(ctx, nil)
	if err != ErrLinkNotFound {
		t.Errorf("GetRandomApprovedLink() error = %v, want ErrLinkNotFound", err)
	}
}
