package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// Test helper functions

// createTestUser creates a test user ID
func createTestUser() identity.UserID {
	return identity.NewUserID()
}

// createTestMetadata creates valid image metadata for testing
func createTestMetadata(title string) gallery.ImageMetadata {
	metadata, err := gallery.NewImageMetadata(
		title,
		"Test description",
		"test-image.jpg",
		"image/jpeg",
		1920, 1080,
		1024*500, // 500KB
		"storage/test/image.jpg",
		"local",
	)
	if err != nil {
		panic(err)
	}
	return metadata
}

// createTestImage creates a test image entity
func createTestImage(ownerID identity.UserID, title string) *gallery.Image {
	metadata := createTestMetadata(title)
	img, err := gallery.NewImage(ownerID, metadata)
	if err != nil {
		panic(err)
	}
	return img
}

// createTestImageWithStatus creates a test image with a specific status
func createTestImageWithStatus(ownerID identity.UserID, title string, status gallery.ImageStatus) *gallery.Image {
	img := createTestImage(ownerID, title)

	// Use domain methods to transition to desired status
	switch status {
	case gallery.StatusActive:
		_ = img.MarkAsActive()
	case gallery.StatusDeleted:
		_ = img.MarkAsDeleted()
	case gallery.StatusFlagged:
		_ = img.Flag()
	// StatusProcessing is default, no action needed
	}

	return img
}

// createTestImageWithVisibility creates a test image with specific visibility
func createTestImageWithVisibility(ownerID identity.UserID, title string, visibility gallery.Visibility) *gallery.Image {
	img := createTestImage(ownerID, title)
	_ = img.MarkAsActive() // Must be active to change visibility
	_ = img.UpdateVisibility(visibility)
	return img
}

// createTestAlbum creates a test album entity
func createTestAlbum(ownerID identity.UserID, title string) *gallery.Album {
	album, err := gallery.NewAlbum(ownerID, title)
	if err != nil {
		panic(err)
	}
	return album
}

// Image Repository Integration Tests

func TestImageRepository_Save_Insert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Test Image")

	// Act
	err = repo.Save(ctx, image)

	// Assert
	require.NoError(t, err)

	// Verify it was saved
	found, err := repo.FindByID(ctx, image.ID())
	require.NoError(t, err)
	assert.Equal(t, image.ID(), found.ID())
	assert.Equal(t, image.OwnerID(), found.OwnerID())
	assert.Equal(t, image.Metadata().Title(), found.Metadata().Title())
	assert.Equal(t, image.Status(), found.Status())
	assert.Equal(t, image.Visibility(), found.Visibility())
}

func TestImageRepository_Save_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Original Title")

	// Save initial version
	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Update metadata
	err = image.UpdateMetadata("Updated Title", "Updated Description")
	require.NoError(t, err)

	// Save updated version
	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(ctx, image.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", found.Metadata().Title())
	assert.Equal(t, "Updated Description", found.Metadata().Description())
}

func TestImageRepository_FindByID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Test Image")

	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Act
	found, err := repo.FindByID(ctx, image.ID())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, image.ID(), found.ID())
	assert.Equal(t, image.OwnerID(), found.OwnerID())
	assert.Equal(t, image.Metadata().Title(), found.Metadata().Title())
}

func TestImageRepository_FindByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	nonExistentID := gallery.NewImageID()

	// Act
	found, err := repo.FindByID(ctx, nonExistentID)

	// Assert
	assert.Nil(t, found)
	require.ErrorIs(t, err, gallery.ErrImageNotFound)
}

func TestImageRepository_FindByOwner_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create 5 images
	for i := 1; i <= 5; i++ {
		img := createTestImage(ownerID, "Image "+string(rune('0'+i)))
		err := repo.Save(ctx, img)
		require.NoError(t, err)
	}

	// Test first page (2 items)
	pagination, _ := shared.NewPagination(1, 2)
	images, total, err := repo.FindByOwner(ctx, ownerID, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, images, 2)

	// Test second page
	pagination, _ = shared.NewPagination(2, 2)
	images, total, err = repo.FindByOwner(ctx, ownerID, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, images, 2)

	// Test third page (only 1 item)
	pagination, _ = shared.NewPagination(3, 2)
	images, total, err = repo.FindByOwner(ctx, ownerID, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, images, 1)
}

func TestImageRepository_FindByOwner_EmptyResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	pagination := shared.DefaultPagination()
	images, total, err := repo.FindByOwner(ctx, ownerID, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, images, 0)
}

func TestImageRepository_FindPublic_OnlyReturnsPublicActiveImages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create public active image (should be returned)
	publicImg := createTestImageWithVisibility(ownerID, "Public Image", gallery.VisibilityPublic)
	err = repo.Save(ctx, publicImg)
	require.NoError(t, err)

	// Create private active image (should NOT be returned)
	privateImg := createTestImageWithVisibility(ownerID, "Private Image", gallery.VisibilityPrivate)
	err = repo.Save(ctx, privateImg)
	require.NoError(t, err)

	// Create public processing image (should NOT be returned - not active)
	processingImg := createTestImage(ownerID, "Processing Image")
	_ = processingImg.UpdateVisibility(gallery.VisibilityPublic) // This will fail but try anyway
	err = repo.Save(ctx, processingImg)
	require.NoError(t, err)

	// Act
	pagination := shared.DefaultPagination()
	images, total, err := repo.FindPublic(ctx, pagination)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, images, 1)
	assert.Equal(t, publicImg.ID(), images[0].ID())
	assert.Equal(t, gallery.StatusActive, images[0].Status())
	assert.Equal(t, gallery.VisibilityPublic, images[0].Visibility())
}

func TestImageRepository_FindPublic_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create 3 public active images
	for i := 1; i <= 3; i++ {
		img := createTestImageWithVisibility(ownerID, "Public Image "+string(rune('0'+i)), gallery.VisibilityPublic)
		err := repo.Save(ctx, img)
		require.NoError(t, err)
	}

	// Test pagination
	pagination, _ := shared.NewPagination(1, 2)
	images, total, err := repo.FindPublic(ctx, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, images, 2)
}

func TestImageRepository_FindByTag_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create tag
	tag := gallery.MustNewTag("nature")

	// Create public active image with tag
	img1 := createTestImageWithVisibility(ownerID, "Nature Image", gallery.VisibilityPublic)
	err = img1.AddTag(tag)
	require.NoError(t, err)
	err = repo.Save(ctx, img1)
	require.NoError(t, err)

	// Create public active image without tag
	img2 := createTestImageWithVisibility(ownerID, "Other Image", gallery.VisibilityPublic)
	err = repo.Save(ctx, img2)
	require.NoError(t, err)

	// Create private image with tag (should NOT be returned)
	img3 := createTestImageWithVisibility(ownerID, "Private Nature", gallery.VisibilityPrivate)
	err = img3.AddTag(tag)
	require.NoError(t, err)
	err = repo.Save(ctx, img3)
	require.NoError(t, err)

	// Act
	pagination := shared.DefaultPagination()
	images, total, err := repo.FindByTag(ctx, tag, pagination)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, images, 1)
	assert.Equal(t, img1.ID(), images[0].ID())
	assert.True(t, images[0].HasTag(tag))
}

func TestImageRepository_FindByTag_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	tag := gallery.MustNewTag("landscape")

	// Create 3 public active images with the tag
	for i := 1; i <= 3; i++ {
		img := createTestImageWithVisibility(ownerID, "Landscape "+string(rune('0'+i)), gallery.VisibilityPublic)
		err := img.AddTag(tag)
		require.NoError(t, err)
		err = repo.Save(ctx, img)
		require.NoError(t, err)
	}

	// Test pagination
	pagination, _ := shared.NewPagination(1, 2)
	images, total, err := repo.FindByTag(ctx, tag, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, images, 2)
}

func TestImageRepository_FindByStatus_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create images with different statuses
	processingImg := createTestImageWithStatus(ownerID, "Processing", gallery.StatusProcessing)
	err = repo.Save(ctx, processingImg)
	require.NoError(t, err)

	activeImg := createTestImageWithStatus(ownerID, "Active", gallery.StatusActive)
	err = repo.Save(ctx, activeImg)
	require.NoError(t, err)

	flaggedImg := createTestImageWithStatus(ownerID, "Flagged", gallery.StatusFlagged)
	err = repo.Save(ctx, flaggedImg)
	require.NoError(t, err)

	// Test finding by status
	tests := []struct {
		name          string
		status        gallery.ImageStatus
		expectedCount int64
		expectedID    gallery.ImageID
	}{
		{"Processing", gallery.StatusProcessing, 1, processingImg.ID()},
		{"Active", gallery.StatusActive, 1, activeImg.ID()},
		{"Flagged", gallery.StatusFlagged, 1, flaggedImg.ID()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pagination := shared.DefaultPagination()
			images, total, err := repo.FindByStatus(ctx, tt.status, pagination)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, total)
			assert.Len(t, images, int(tt.expectedCount))
			if len(images) > 0 {
				assert.Equal(t, tt.expectedID, images[0].ID())
				assert.Equal(t, tt.status, images[0].Status())
			}
		})
	}
}

func TestImageRepository_Delete_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Test Image")

	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Act
	err = repo.Delete(ctx, image.ID())

	// Assert
	require.NoError(t, err)

	// Verify it's deleted
	found, err := repo.FindByID(ctx, image.ID())
	assert.Nil(t, found)
	require.ErrorIs(t, err, gallery.ErrImageNotFound)
}

func TestImageRepository_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	nonExistentID := gallery.NewImageID()

	// Act
	err = repo.Delete(ctx, nonExistentID)

	// Assert
	require.ErrorIs(t, err, gallery.ErrImageNotFound)
}

func TestImageRepository_ExistsByID_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Test Image")

	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Act
	exists, err := repo.ExistsByID(ctx, image.ID())

	// Assert
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestImageRepository_ExistsByID_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	nonExistentID := gallery.NewImageID()

	// Act
	exists, err := repo.ExistsByID(ctx, nonExistentID)

	// Assert
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestImageRepository_SaveWithTags_PreservesTagsOnUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Test Image")

	// Add tags
	tag1 := gallery.MustNewTag("nature")
	tag2 := gallery.MustNewTag("landscape")
	err = image.AddTag(tag1)
	require.NoError(t, err)
	err = image.AddTag(tag2)
	require.NoError(t, err)

	// Save with tags
	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Verify tags are saved
	found, err := repo.FindByID(ctx, image.ID())
	require.NoError(t, err)
	assert.Len(t, found.Tags(), 2)
	assert.True(t, found.HasTag(tag1))
	assert.True(t, found.HasTag(tag2))

	// Update image (change metadata)
	err = image.UpdateMetadata("Updated Title", "Updated Description")
	require.NoError(t, err)

	// Save update
	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Verify tags are still there
	found, err = repo.FindByID(ctx, image.ID())
	require.NoError(t, err)
	assert.Len(t, found.Tags(), 2)
	assert.True(t, found.HasTag(tag1))
	assert.True(t, found.HasTag(tag2))
}

func TestImageRepository_SaveWithVariants_PreservesVariantsOnUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	image := createTestImage(ownerID, "Test Image")

	// Add variants
	thumbnail, err := gallery.NewImageVariant(
		gallery.VariantThumbnail,
		"storage/thumb.jpg",
		160, 120,
		5000,
		"jpeg",
	)
	require.NoError(t, err)
	err = image.AddVariant(thumbnail)
	require.NoError(t, err)

	small, err := gallery.NewImageVariant(
		gallery.VariantSmall,
		"storage/small.jpg",
		320, 240,
		15000,
		"jpeg",
	)
	require.NoError(t, err)
	err = image.AddVariant(small)
	require.NoError(t, err)

	// Save with variants
	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Verify variants are saved
	found, err := repo.FindByID(ctx, image.ID())
	require.NoError(t, err)
	assert.Len(t, found.Variants(), 2)
	assert.True(t, found.HasVariant(gallery.VariantThumbnail))
	assert.True(t, found.HasVariant(gallery.VariantSmall))

	// Update image (change metadata)
	err = image.UpdateMetadata("Updated Title", "Updated Description")
	require.NoError(t, err)

	// Save update
	err = repo.Save(ctx, image)
	require.NoError(t, err)

	// Verify variants are still there
	found, err = repo.FindByID(ctx, image.ID())
	require.NoError(t, err)
	assert.Len(t, found.Variants(), 2)
}

// Album Repository Integration Tests

func TestAlbumRepository_Save_Insert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()
	album := createTestAlbum(ownerID, "Test Album")

	// Act
	err = repo.Save(ctx, album)

	// Assert
	require.NoError(t, err)

	// Verify it was saved
	found, err := repo.FindByID(ctx, album.ID())
	require.NoError(t, err)
	assert.Equal(t, album.ID(), found.ID())
	assert.Equal(t, album.OwnerID(), found.OwnerID())
	assert.Equal(t, album.Title(), found.Title())
	assert.Equal(t, album.Visibility(), found.Visibility())
}

func TestAlbumRepository_Save_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()
	album := createTestAlbum(ownerID, "Original Title")

	// Save initial version
	err = repo.Save(ctx, album)
	require.NoError(t, err)

	// Update album
	err = album.UpdateTitle("Updated Title")
	require.NoError(t, err)
	err = album.UpdateDescription("Updated Description")
	require.NoError(t, err)

	// Save updated version
	err = repo.Save(ctx, album)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(ctx, album.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", found.Title())
	assert.Equal(t, "Updated Description", found.Description())
}

func TestAlbumRepository_FindByID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()
	album := createTestAlbum(ownerID, "Test Album")

	err = repo.Save(ctx, album)
	require.NoError(t, err)

	// Act
	found, err := repo.FindByID(ctx, album.ID())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, album.ID(), found.ID())
	assert.Equal(t, album.OwnerID(), found.OwnerID())
	assert.Equal(t, album.Title(), found.Title())
}

func TestAlbumRepository_FindByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	nonExistentID := gallery.NewAlbumID()

	// Act
	found, err := repo.FindByID(ctx, nonExistentID)

	// Assert
	assert.Nil(t, found)
	require.ErrorIs(t, err, gallery.ErrAlbumNotFound)
}

func TestAlbumRepository_FindByOwner_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create 3 albums
	for i := 1; i <= 3; i++ {
		album := createTestAlbum(ownerID, "Album "+string(rune('0'+i)))
		err := repo.Save(ctx, album)
		require.NoError(t, err)
	}

	// Act
	albums, err := repo.FindByOwner(ctx, ownerID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, albums, 3)
}

func TestAlbumRepository_FindByOwner_EmptyResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Act
	albums, err := repo.FindByOwner(ctx, ownerID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, albums, 0)
}

func TestAlbumRepository_FindPublic_OnlyReturnsPublicAlbums(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create public album
	publicAlbum := createTestAlbum(ownerID, "Public Album")
	err = publicAlbum.UpdateVisibility(gallery.VisibilityPublic)
	require.NoError(t, err)
	err = repo.Save(ctx, publicAlbum)
	require.NoError(t, err)

	// Create private album
	privateAlbum := createTestAlbum(ownerID, "Private Album")
	err = repo.Save(ctx, privateAlbum)
	require.NoError(t, err)

	// Act
	pagination := shared.DefaultPagination()
	albums, total, err := repo.FindPublic(ctx, pagination)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, albums, 1)
	assert.Equal(t, publicAlbum.ID(), albums[0].ID())
	assert.Equal(t, gallery.VisibilityPublic, albums[0].Visibility())
}

func TestAlbumRepository_FindPublic_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create 3 public albums
	for i := 1; i <= 3; i++ {
		album := createTestAlbum(ownerID, "Public Album "+string(rune('0'+i)))
		err := album.UpdateVisibility(gallery.VisibilityPublic)
		require.NoError(t, err)
		err = repo.Save(ctx, album)
		require.NoError(t, err)
	}

	// Test pagination
	pagination, _ := shared.NewPagination(1, 2)
	albums, total, err := repo.FindPublic(ctx, pagination)

	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, albums, 2)
}

func TestAlbumRepository_Delete_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()
	album := createTestAlbum(ownerID, "Test Album")

	err = repo.Save(ctx, album)
	require.NoError(t, err)

	// Act
	err = repo.Delete(ctx, album.ID())

	// Assert
	require.NoError(t, err)

	// Verify it's deleted
	found, err := repo.FindByID(ctx, album.ID())
	assert.Nil(t, found)
	require.ErrorIs(t, err, gallery.ErrAlbumNotFound)
}

func TestAlbumRepository_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	nonExistentID := gallery.NewAlbumID()

	// Act
	err = repo.Delete(ctx, nonExistentID)

	// Assert
	require.ErrorIs(t, err, gallery.ErrAlbumNotFound)
}

func TestAlbumRepository_ExistsByID_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()
	album := createTestAlbum(ownerID, "Test Album")

	err = repo.Save(ctx, album)
	require.NoError(t, err)

	// Act
	exists, err := repo.ExistsByID(ctx, album.ID())

	// Assert
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestAlbumRepository_ExistsByID_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewAlbumRepository(pgContainer.DB)
	nonExistentID := gallery.NewAlbumID()

	// Act
	exists, err := repo.ExistsByID(ctx, nonExistentID)

	// Assert
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestAlbumRepository_SaveWithCoverImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	imageRepo := postgres.NewImageRepository(pgContainer.DB)
	albumRepo := postgres.NewAlbumRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create and save an image
	image := createTestImage(ownerID, "Cover Image")
	err = imageRepo.Save(ctx, image)
	require.NoError(t, err)

	// Create album with cover image
	album := createTestAlbum(ownerID, "Test Album")
	imageID := image.ID()
	album.SetCoverImage(&imageID)

	// Save album
	err = albumRepo.Save(ctx, album)
	require.NoError(t, err)

	// Verify cover image is saved
	found, err := albumRepo.FindByID(ctx, album.ID())
	require.NoError(t, err)
	assert.NotNil(t, found.CoverImageID())
	assert.Equal(t, image.ID(), *found.CoverImageID())

	// Remove cover image
	album.SetCoverImage(nil)
	err = albumRepo.Save(ctx, album)
	require.NoError(t, err)

	// Verify cover image is removed
	found, err = albumRepo.FindByID(ctx, album.ID())
	require.NoError(t, err)
	assert.Nil(t, found.CoverImageID())
}
