package db

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func TestCreateOrganization(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	org := &models.Organization{
		Name: "Test Organization",
		Slug: "test-organization",
	}

	err := db.CreateOrganization(ctx, org)
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	if org.ID == uuid.Nil {
		t.Error("CreateOrganization() did not set ID")
	}
	if org.CreatedAt.IsZero() {
		t.Error("CreateOrganization() did not set CreatedAt")
	}
}

func TestCreateOrganization_DuplicateSlug(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	org1 := &models.Organization{
		Name: "Org One",
		Slug: "duplicate-slug",
	}
	if err := db.CreateOrganization(ctx, org1); err != nil {
		t.Fatalf("CreateOrganization() first org error = %v", err)
	}

	org2 := &models.Organization{
		Name: "Org Two",
		Slug: "duplicate-slug",
	}
	err := db.CreateOrganization(ctx, org2)
	if err == nil {
		t.Error("CreateOrganization() should fail with duplicate slug")
	}
}

func TestGetOrganizationByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	org := &models.Organization{
		Name: "Get By ID Org",
		Slug: "get-by-id-org",
	}
	if err := db.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	found, err := db.GetOrganizationByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetOrganizationByID() error = %v", err)
	}
	if found.Name != "Get By ID Org" {
		t.Errorf("GetOrganizationByID() name = %q, want %q", found.Name, "Get By ID Org")
	}

	// Not found
	_, err = db.GetOrganizationByID(ctx, uuid.New())
	if err != ErrOrgNotFound {
		t.Errorf("GetOrganizationByID() error = %v, want ErrOrgNotFound", err)
	}
}

func TestGetOrganizationBySlug(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	org := &models.Organization{
		Name: "Get By Slug Org",
		Slug: "get-by-slug-org",
	}
	if err := db.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	found, err := db.GetOrganizationBySlug(ctx, "get-by-slug-org")
	if err != nil {
		t.Fatalf("GetOrganizationBySlug() error = %v", err)
	}
	if found.Name != "Get By Slug Org" {
		t.Errorf("GetOrganizationBySlug() name = %q, want %q", found.Name, "Get By Slug Org")
	}

	// Not found
	_, err = db.GetOrganizationBySlug(ctx, "non-existent")
	if err != ErrOrgNotFound {
		t.Errorf("GetOrganizationBySlug() error = %v, want ErrOrgNotFound", err)
	}
}

func TestGetAllOrganizations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple orgs
	orgs := []struct {
		name string
		slug string
	}{
		{"Alpha Org", "alpha-org"},
		{"Beta Org", "beta-org"},
		{"Charlie Org", "charlie-org"},
	}
	for _, o := range orgs {
		org := &models.Organization{Name: o.name, Slug: o.slug}
		if err := db.CreateOrganization(ctx, org); err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}
	}

	all, err := db.GetAllOrganizations(ctx)
	if err != nil {
		t.Fatalf("GetAllOrganizations() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("GetAllOrganizations() returned %d orgs, want 3", len(all))
	}

	// Should be sorted by name
	if all[0].Name != "Alpha Org" {
		t.Errorf("GetAllOrganizations() first org = %q, want %q", all[0].Name, "Alpha Org")
	}
}
