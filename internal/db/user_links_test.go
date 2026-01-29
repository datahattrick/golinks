package db

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func TestCreateUserLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Sub:   "userlink-sub-123",
		Email: "userlink@example.com",
		Name:  "User Link User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	userLink := &models.UserLink{
		UserID:      user.ID,
		Keyword:     "my-personal-link",
		URL:         "https://personal.example.com",
		Description: "Personal link",
	}

	err := db.CreateUserLink(ctx, userLink)
	if err != nil {
		t.Fatalf("CreateUserLink() error = %v", err)
	}

	if userLink.ID == uuid.Nil {
		t.Error("CreateUserLink() did not set ID")
	}
}

func TestCreateUserLink_DuplicateKeyword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "dup-userlink-sub",
		Email: "dup-userlink@example.com",
		Name:  "Dup User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link1 := &models.UserLink{
		UserID:  user.ID,
		Keyword: "duplicate",
		URL:     "https://first.example.com",
	}
	if err := db.CreateUserLink(ctx, link1); err != nil {
		t.Fatalf("CreateUserLink() first link error = %v", err)
	}

	link2 := &models.UserLink{
		UserID:  user.ID,
		Keyword: "duplicate",
		URL:     "https://second.example.com",
	}
	err := db.CreateUserLink(ctx, link2)
	if err != ErrDuplicateKeyword {
		t.Errorf("CreateUserLink() error = %v, want ErrDuplicateKeyword", err)
	}
}

func TestGetUserLinkByKeyword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "get-userlink-sub",
		Email: "get-userlink@example.com",
		Name:  "Get User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.UserLink{
		UserID:  user.ID,
		Keyword: "get-link",
		URL:     "https://get.example.com",
	}
	if err := db.CreateUserLink(ctx, link); err != nil {
		t.Fatalf("CreateUserLink() error = %v", err)
	}

	found, err := db.GetUserLinkByKeyword(ctx, user.ID, "get-link")
	if err != nil {
		t.Fatalf("GetUserLinkByKeyword() error = %v", err)
	}
	if found.URL != "https://get.example.com" {
		t.Errorf("GetUserLinkByKeyword() URL = %q, want %q", found.URL, "https://get.example.com")
	}

	// Not found
	_, err = db.GetUserLinkByKeyword(ctx, user.ID, "non-existent")
	if err != ErrUserLinkNotFound {
		t.Errorf("GetUserLinkByKeyword() error = %v, want ErrUserLinkNotFound", err)
	}

	// Other user shouldn't find it
	otherUser := &models.User{
		Sub:   "other-user-sub",
		Email: "other@example.com",
		Name:  "Other User",
	}
	if err := db.UpsertUser(ctx, otherUser); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}
	_, err = db.GetUserLinkByKeyword(ctx, otherUser.ID, "get-link")
	if err != ErrUserLinkNotFound {
		t.Errorf("GetUserLinkByKeyword() other user error = %v, want ErrUserLinkNotFound", err)
	}
}

func TestGetUserLinks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "list-userlink-sub",
		Email: "list-userlink@example.com",
		Name:  "List User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	// Create multiple links
	keywords := []string{"aaa-link", "bbb-link", "ccc-link"}
	for _, kw := range keywords {
		link := &models.UserLink{
			UserID:  user.ID,
			Keyword: kw,
			URL:     "https://example.com/" + kw,
		}
		if err := db.CreateUserLink(ctx, link); err != nil {
			t.Fatalf("CreateUserLink() error = %v", err)
		}
	}

	links, err := db.GetUserLinks(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserLinks() error = %v", err)
	}
	if len(links) != 3 {
		t.Errorf("GetUserLinks() returned %d links, want 3", len(links))
	}

	// Should be sorted by keyword
	if links[0].Keyword != "aaa-link" {
		t.Errorf("GetUserLinks() first link = %q, want %q", links[0].Keyword, "aaa-link")
	}
}

func TestUpdateUserLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "update-userlink-sub",
		Email: "update-userlink@example.com",
		Name:  "Update User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.UserLink{
		UserID:      user.ID,
		Keyword:     "update-link",
		URL:         "https://original.example.com",
		Description: "Original",
	}
	if err := db.CreateUserLink(ctx, link); err != nil {
		t.Fatalf("CreateUserLink() error = %v", err)
	}

	link.URL = "https://updated.example.com"
	link.Description = "Updated"
	if err := db.UpdateUserLink(ctx, link); err != nil {
		t.Fatalf("UpdateUserLink() error = %v", err)
	}

	updated, err := db.GetUserLinkByKeyword(ctx, user.ID, "update-link")
	if err != nil {
		t.Fatalf("GetUserLinkByKeyword() error = %v", err)
	}
	if updated.URL != "https://updated.example.com" {
		t.Errorf("UpdateUserLink() URL = %q, want %q", updated.URL, "https://updated.example.com")
	}
	if updated.Description != "Updated" {
		t.Errorf("UpdateUserLink() description = %q, want %q", updated.Description, "Updated")
	}
}

func TestDeleteUserLink(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "delete-userlink-sub",
		Email: "delete-userlink@example.com",
		Name:  "Delete User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.UserLink{
		UserID:  user.ID,
		Keyword: "delete-link",
		URL:     "https://delete.example.com",
	}
	if err := db.CreateUserLink(ctx, link); err != nil {
		t.Fatalf("CreateUserLink() error = %v", err)
	}

	if err := db.DeleteUserLink(ctx, link.ID, user.ID); err != nil {
		t.Fatalf("DeleteUserLink() error = %v", err)
	}

	// Verify deleted
	_, err := db.GetUserLinkByKeyword(ctx, user.ID, "delete-link")
	if err != ErrUserLinkNotFound {
		t.Errorf("GetUserLinkByKeyword() after delete error = %v, want ErrUserLinkNotFound", err)
	}
}

func TestIncrementUserLinkClickCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "click-userlink-sub",
		Email: "click-userlink@example.com",
		Name:  "Click User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	link := &models.UserLink{
		UserID:  user.ID,
		Keyword: "click-link",
		URL:     "https://click.example.com",
	}
	if err := db.CreateUserLink(ctx, link); err != nil {
		t.Fatalf("CreateUserLink() error = %v", err)
	}

	// Increment multiple times
	for i := 0; i < 3; i++ {
		if err := db.IncrementUserLinkClickCount(ctx, user.ID, "click-link"); err != nil {
			t.Fatalf("IncrementUserLinkClickCount() error = %v", err)
		}
	}

	updated, err := db.GetUserLinkByKeyword(ctx, user.ID, "click-link")
	if err != nil {
		t.Fatalf("GetUserLinkByKeyword() error = %v", err)
	}
	if updated.ClickCount != 3 {
		t.Errorf("IncrementUserLinkClickCount() click_count = %d, want 3", updated.ClickCount)
	}
}
