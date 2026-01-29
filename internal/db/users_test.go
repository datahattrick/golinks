package db

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func TestUpsertUser_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "test-sub-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  models.RoleUser,
	}

	err := db.UpsertUser(ctx, user)
	if err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("UpsertUser() did not set ID")
	}
	if user.Role != models.RoleUser {
		t.Errorf("UpsertUser() role = %q, want %q", user.Role, models.RoleUser)
	}
}

func TestUpsertUser_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create user
	user := &models.User{
		Sub:   "update-sub-123",
		Email: "original@example.com",
		Name:  "Original Name",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() create error = %v", err)
	}
	originalID := user.ID

	// Update user
	user.Email = "updated@example.com"
	user.Name = "Updated Name"
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() update error = %v", err)
	}

	// ID should be the same
	if user.ID != originalID {
		t.Errorf("UpsertUser() changed ID from %v to %v", originalID, user.ID)
	}

	// Verify update
	fetched, err := db.GetUserBySub(ctx, "update-sub-123")
	if err != nil {
		t.Fatalf("GetUserBySub() error = %v", err)
	}
	if fetched.Email != "updated@example.com" {
		t.Errorf("UpsertUser() email = %q, want %q", fetched.Email, "updated@example.com")
	}
	if fetched.Name != "Updated Name" {
		t.Errorf("UpsertUser() name = %q, want %q", fetched.Name, "Updated Name")
	}
}

func TestGetUserBySub(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create user
	user := &models.User{
		Sub:   "get-sub-123",
		Email: "get@example.com",
		Name:  "Get User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	// Find by sub
	found, err := db.GetUserBySub(ctx, "get-sub-123")
	if err != nil {
		t.Fatalf("GetUserBySub() error = %v", err)
	}
	if found.Email != "get@example.com" {
		t.Errorf("GetUserBySub() email = %q, want %q", found.Email, "get@example.com")
	}

	// Not found
	_, err = db.GetUserBySub(ctx, "non-existent")
	if err != ErrUserNotFound {
		t.Errorf("GetUserBySub() error = %v, want ErrUserNotFound", err)
	}
}

func TestGetUserByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "id-sub-123",
		Email: "id@example.com",
		Name:  "ID User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	found, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if found.Sub != "id-sub-123" {
		t.Errorf("GetUserByID() sub = %q, want %q", found.Sub, "id-sub-123")
	}

	// Not found
	_, err = db.GetUserByID(ctx, uuid.New())
	if err != ErrUserNotFound {
		t.Errorf("GetUserByID() error = %v, want ErrUserNotFound", err)
	}
}

func TestGetUserByUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:      "username-sub-123",
		Username: "testuser",
		Email:    "username@example.com",
		Name:     "Username User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	found, err := db.GetUserByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if found.Sub != "username-sub-123" {
		t.Errorf("GetUserByUsername() sub = %q, want %q", found.Sub, "username-sub-123")
	}
}

func TestUpdateUserRole(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Sub:   "role-sub-123",
		Email: "role@example.com",
		Name:  "Role User",
		Role:  models.RoleUser,
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	// Update role to admin
	if err := db.UpdateUserRole(ctx, user.ID, models.RoleAdmin); err != nil {
		t.Fatalf("UpdateUserRole() error = %v", err)
	}

	updated, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if updated.Role != models.RoleAdmin {
		t.Errorf("UpdateUserRole() role = %q, want %q", updated.Role, models.RoleAdmin)
	}
}

func TestUpdateUserOrganization(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create organization
	org := &models.Organization{
		Name: "Test Org",
		Slug: "test-org",
	}
	if err := db.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	// Create user
	user := &models.User{
		Sub:   "org-sub-123",
		Email: "org@example.com",
		Name:  "Org User",
	}
	if err := db.UpsertUser(ctx, user); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	// Assign to organization
	if err := db.UpdateUserOrganization(ctx, user.ID, &org.ID); err != nil {
		t.Fatalf("UpdateUserOrganization() error = %v", err)
	}

	updated, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if updated.OrganizationID == nil || *updated.OrganizationID != org.ID {
		t.Error("UpdateUserOrganization() did not set organization")
	}

	// Remove from organization
	if err := db.UpdateUserOrganization(ctx, user.ID, nil); err != nil {
		t.Fatalf("UpdateUserOrganization(nil) error = %v", err)
	}

	updated, err = db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if updated.OrganizationID != nil {
		t.Error("UpdateUserOrganization(nil) did not remove organization")
	}
}
