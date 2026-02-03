package main

import (
	"context"
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// TestNoOpNSFWScanRepository_SingularMethods verifies that singular find methods
// return ErrNSFWScanNotFound as expected.
func TestNoOpNSFWScanRepository_SingularMethods(t *testing.T) {
	repo := &noOpNSFWScanRepository{}
	ctx := context.Background()

	t.Run("FindByID returns ErrNSFWScanNotFound", func(t *testing.T) {
		scan, err := repo.FindByID(ctx, moderation.NSFWScanID{})
		if err != moderation.ErrNSFWScanNotFound {
			t.Errorf("Expected ErrNSFWScanNotFound, got %v", err)
		}
		if scan != nil {
			t.Errorf("Expected nil scan, got %v", scan)
		}
	})

	t.Run("FindByImageID returns ErrNSFWScanNotFound", func(t *testing.T) {
		scan, err := repo.FindByImageID(ctx, gallery.ImageID{})
		if err != moderation.ErrNSFWScanNotFound {
			t.Errorf("Expected ErrNSFWScanNotFound, got %v", err)
		}
		if scan != nil {
			t.Errorf("Expected nil scan, got %v", scan)
		}
	})
}

// TestNoOpNSFWScanRepository_CollectionMethods verifies that collection methods
// return empty slices with nil error, not errors.
func TestNoOpNSFWScanRepository_CollectionMethods(t *testing.T) {
	repo := &noOpNSFWScanRepository{}
	ctx := context.Background()
	pagination, _ := shared.NewPagination(1, 10)

	t.Run("FindByImageIDAll returns empty slice", func(t *testing.T) {
		scans, err := repo.FindByImageIDAll(ctx, gallery.ImageID{})
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if scans == nil {
			t.Error("Expected non-nil slice, got nil")
		}
		if len(scans) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(scans))
		}
	})

	t.Run("FindPending returns empty slice", func(t *testing.T) {
		scans, total, err := repo.FindPending(ctx, pagination)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if scans == nil {
			t.Error("Expected non-nil slice, got nil")
		}
		if len(scans) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(scans))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("FindByStatus returns empty slice", func(t *testing.T) {
		scans, total, err := repo.FindByStatus(ctx, moderation.ScanStatusPending, pagination)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if scans == nil {
			t.Error("Expected non-nil slice, got nil")
		}
		if len(scans) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(scans))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("FindNSFWImages returns empty slice", func(t *testing.T) {
		scans, total, err := repo.FindNSFWImages(ctx, pagination)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if scans == nil {
			t.Error("Expected non-nil slice, got nil")
		}
		if len(scans) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(scans))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})
}

// TestNoOpNSFWScanRepository_OtherMethods verifies other stub methods work as expected.
func TestNoOpNSFWScanRepository_OtherMethods(t *testing.T) {
	repo := &noOpNSFWScanRepository{}
	ctx := context.Background()

	t.Run("HasActiveScan returns false", func(t *testing.T) {
		active, err := repo.HasActiveScan(ctx, gallery.ImageID{})
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if active {
			t.Error("Expected false, got true")
		}
	})

	t.Run("Save returns nil", func(t *testing.T) {
		err := repo.Save(ctx, &moderation.NSFWScan{})
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("NextID returns empty ID", func(t *testing.T) {
		id := repo.NextID()
		if id != (moderation.NSFWScanID{}) {
			t.Errorf("Expected empty ID, got %v", id)
		}
	})
}
