package service

import (
	"context"
	"testing"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

// fakeDevelopmentRepository is an in-memory stand-in for domain.DevelopmentRepository
// so PortfolioService's business rules (slug conflicts, publish transitions,
// status validation) can be tested without a database.
type fakeDevelopmentRepository struct {
	byID   map[string]*domain.Development
	nextID int
}

func newFakeDevelopmentRepository() *fakeDevelopmentRepository {
	return &fakeDevelopmentRepository{byID: make(map[string]*domain.Development)}
}

func (f *fakeDevelopmentRepository) Create(ctx context.Context, d *domain.Development) error {
	f.nextID++
	d.ID = string(rune('a' + f.nextID))
	cp := *d
	f.byID[d.ID] = &cp
	return nil
}

func (f *fakeDevelopmentRepository) Update(ctx context.Context, d *domain.Development) error {
	if _, ok := f.byID[d.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *d
	f.byID[d.ID] = &cp
	return nil
}

func (f *fakeDevelopmentRepository) Delete(ctx context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeDevelopmentRepository) GetByID(ctx context.Context, id string) (*domain.Development, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (f *fakeDevelopmentRepository) GetBySlug(ctx context.Context, slug string) (*domain.Development, error) {
	for _, d := range f.byID {
		if d.Slug == slug {
			cp := *d
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeDevelopmentRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Development], error) {
	var items []domain.Development
	for _, d := range f.byID {
		items = append(items, *d)
	}
	return domain.PageResult[domain.Development]{Items: items, TotalItems: len(items), Page: params.Page, PerPage: params.PerPage}, nil
}

func (f *fakeDevelopmentRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	for _, d := range f.byID {
		if d.Slug == slug && d.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

type noopRevalidator struct{}

func (noopRevalidator) Trigger(ctx context.Context, tag string) {}

func TestPortfolioService_Create_RejectsInvalidStatus(t *testing.T) {
	svc := NewPortfolioService(newFakeDevelopmentRepository(), noopRevalidator{})

	_, err := svc.Create(context.Background(), &domain.Development{Slug: "test", Status: "not-a-real-status"})

	appErr := asAppError(t, err)
	if appErr.Code != apperror.CodeValidation {
		t.Fatalf("expected validation error, got %q", appErr.Code)
	}
}

func TestPortfolioService_Create_RejectsDuplicateSlug(t *testing.T) {
	repo := newFakeDevelopmentRepository()
	svc := NewPortfolioService(repo, noopRevalidator{})
	ctx := context.Background()

	if _, err := svc.Create(ctx, &domain.Development{Slug: "seren-redwood", Status: domain.DevelopmentStatusPlanning}); err != nil {
		t.Fatalf("first create should succeed, got %v", err)
	}

	_, err := svc.Create(ctx, &domain.Development{Slug: "seren-redwood", Status: domain.DevelopmentStatusPlanning})
	appErr := asAppError(t, err)
	if appErr.Code != apperror.CodeConflict {
		t.Fatalf("expected conflict error, got %q", appErr.Code)
	}
}

func TestPortfolioService_Create_SetsPublishedAtWhenPublished(t *testing.T) {
	svc := NewPortfolioService(newFakeDevelopmentRepository(), noopRevalidator{})

	created, err := svc.Create(context.Background(), &domain.Development{
		Slug: "seren-redwood", Status: domain.DevelopmentStatusPlanning, Published: true,
	})
	if err != nil {
		t.Fatalf("create should succeed, got %v", err)
	}
	if created.PublishedAt == nil {
		t.Fatal("expected PublishedAt to be set when Published is true")
	}
}

func TestPortfolioService_Create_LeavesPublishedAtNilWhenUnpublished(t *testing.T) {
	svc := NewPortfolioService(newFakeDevelopmentRepository(), noopRevalidator{})

	created, err := svc.Create(context.Background(), &domain.Development{
		Slug: "seren-redwood", Status: domain.DevelopmentStatusPlanning, Published: false,
	})
	if err != nil {
		t.Fatalf("create should succeed, got %v", err)
	}
	if created.PublishedAt != nil {
		t.Fatal("expected PublishedAt to stay nil when Published is false")
	}
}

func TestPortfolioService_Update_PreservesPublishedAtOnceSet(t *testing.T) {
	repo := newFakeDevelopmentRepository()
	svc := NewPortfolioService(repo, noopRevalidator{})
	ctx := context.Background()

	created, err := svc.Create(ctx, &domain.Development{
		Slug: "seren-redwood", Status: domain.DevelopmentStatusPlanning, Published: true,
	})
	if err != nil {
		t.Fatalf("create should succeed, got %v", err)
	}
	firstPublishedAt := created.PublishedAt

	created.Status = domain.DevelopmentStatusUnderConstruction
	updated, err := svc.Update(ctx, created)
	if err != nil {
		t.Fatalf("update should succeed, got %v", err)
	}
	if updated.PublishedAt == nil || !updated.PublishedAt.Equal(*firstPublishedAt) {
		t.Fatalf("expected PublishedAt to stay stable across an update, got %v want %v", updated.PublishedAt, firstPublishedAt)
	}
}

func TestPortfolioService_GetBySlug_NotFound(t *testing.T) {
	svc := NewPortfolioService(newFakeDevelopmentRepository(), noopRevalidator{})

	_, err := svc.GetBySlug(context.Background(), "does-not-exist")
	appErr := asAppError(t, err)
	if appErr.Code != apperror.CodeNotFound {
		t.Fatalf("expected not-found error, got %q", appErr.Code)
	}
}

func asAppError(t *testing.T, err error) *apperror.Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	appErr, ok := err.(*apperror.Error)
	if !ok {
		t.Fatalf("expected *apperror.Error, got %T", err)
	}
	return appErr
}
