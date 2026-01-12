package commands_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/community/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/community/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestCreateGroupHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockGroupRepo := new(testhelpers.MockGroupRepository)
	mockMembershipRepo := new(testhelpers.MockGroupMembershipRepository)
	mockPublisher := new(testhelpers.MockEventPublisher)
	logger := zerolog.Nop()

	handler := commands.NewCreateGroupHandler(mockGroupRepo, mockMembershipRepo, mockPublisher, &logger)

	ownerID := identity.NewUserID()
	cmd := commands.CreateGroupCommand{
		OwnerID:     ownerID,
		Name:        "Photography Enthusiasts",
		Slug:        "photography-enthusiasts",
		Description: "A group for photography lovers",
		GroupType:   "public",
		Settings:    nil, // Use defaults
	}

	slug, _ := community.NewGroupSlug(cmd.Slug)

	// Mock expectations
	mockGroupRepo.On("ExistsWithSlug", mock.Anything, slug).Return(false, nil)
	mockGroupRepo.On("Save", mock.Anything, mock.AnythingOfType("*community.Group")).Return(nil)
	mockMembershipRepo.On("Save", mock.Anything, mock.AnythingOfType("*community.GroupMembership")).Return(nil)
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	// Act
	group, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, ownerID, group.OwnerID())
	assert.Equal(t, "Photography Enthusiasts", group.Name().String())
	assert.Equal(t, "photography-enthusiasts", group.Slug().String())
	assert.Equal(t, "A group for photography lovers", group.Description())
	assert.Equal(t, community.GroupTypePublic, group.GroupType())

	mockGroupRepo.AssertExpectations(t)
	mockMembershipRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCreateGroupHandler_Handle_DuplicateSlug(t *testing.T) {
	t.Parallel()

	// Arrange
	mockGroupRepo := new(testhelpers.MockGroupRepository)
	mockMembershipRepo := new(testhelpers.MockGroupMembershipRepository)
	mockPublisher := new(testhelpers.MockEventPublisher)
	logger := zerolog.Nop()

	handler := commands.NewCreateGroupHandler(mockGroupRepo, mockMembershipRepo, mockPublisher, &logger)

	ownerID := identity.NewUserID()
	cmd := commands.CreateGroupCommand{
		OwnerID:   ownerID,
		Name:      "Photography Enthusiasts",
		Slug:      "photography-enthusiasts",
		GroupType: "public",
	}

	slug, _ := community.NewGroupSlug(cmd.Slug)

	// Mock expectations - slug already exists
	mockGroupRepo.On("ExistsWithSlug", mock.Anything, slug).Return(true, nil)

	// Act
	group, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Nil(t, group)
	require.ErrorIs(t, err, community.ErrGroupSlugTaken)

	mockGroupRepo.AssertExpectations(t)
	mockMembershipRepo.AssertNotCalled(t, "Save")
	mockPublisher.AssertNotCalled(t, "Publish")
}

func TestCreateGroupHandler_Handle_InvalidName(t *testing.T) {
	t.Parallel()

	// Arrange
	mockGroupRepo := new(testhelpers.MockGroupRepository)
	mockMembershipRepo := new(testhelpers.MockGroupMembershipRepository)
	mockPublisher := new(testhelpers.MockEventPublisher)
	logger := zerolog.Nop()

	handler := commands.NewCreateGroupHandler(mockGroupRepo, mockMembershipRepo, mockPublisher, &logger)

	ownerID := identity.NewUserID()
	cmd := commands.CreateGroupCommand{
		OwnerID:   ownerID,
		Name:      "ab", // Too short
		Slug:      "test-group",
		GroupType: "public",
	}

	// Act
	group, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Nil(t, group)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid group name")

	mockGroupRepo.AssertNotCalled(t, "ExistsWithSlug")
	mockMembershipRepo.AssertNotCalled(t, "Save")
	mockPublisher.AssertNotCalled(t, "Publish")
}

func TestCreateGroupHandler_Handle_InvalidGroupType(t *testing.T) {
	t.Parallel()

	// Arrange
	mockGroupRepo := new(testhelpers.MockGroupRepository)
	mockMembershipRepo := new(testhelpers.MockGroupMembershipRepository)
	mockPublisher := new(testhelpers.MockEventPublisher)
	logger := zerolog.Nop()

	handler := commands.NewCreateGroupHandler(mockGroupRepo, mockMembershipRepo, mockPublisher, &logger)

	ownerID := identity.NewUserID()
	cmd := commands.CreateGroupCommand{
		OwnerID:   ownerID,
		Name:      "Test Group",
		Slug:      "test-group",
		GroupType: "invalid", // Invalid type
	}

	// Act
	group, err := handler.Handle(context.Background(), cmd)

	// Assert
	assert.Nil(t, group)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid group type")

	mockGroupRepo.AssertNotCalled(t, "ExistsWithSlug")
	mockMembershipRepo.AssertNotCalled(t, "Save")
	mockPublisher.AssertNotCalled(t, "Publish")
}

func TestCreateGroupHandler_Handle_WithCustomSettings(t *testing.T) {
	t.Parallel()

	// Arrange
	mockGroupRepo := new(testhelpers.MockGroupRepository)
	mockMembershipRepo := new(testhelpers.MockGroupMembershipRepository)
	mockPublisher := new(testhelpers.MockEventPublisher)
	logger := zerolog.Nop()

	handler := commands.NewCreateGroupHandler(mockGroupRepo, mockMembershipRepo, mockPublisher, &logger)

	ownerID := identity.NewUserID()
	settings, _ := community.NewGroupSettings(true, true, true, 100)
	cmd := commands.CreateGroupCommand{
		OwnerID:   ownerID,
		Name:      "Limited Group",
		Slug:      "limited-group",
		GroupType: "invite_only",
		Settings:  &settings,
	}

	slug, _ := community.NewGroupSlug(cmd.Slug)

	// Mock expectations
	mockGroupRepo.On("ExistsWithSlug", mock.Anything, slug).Return(false, nil)
	mockGroupRepo.On("Save", mock.Anything, mock.AnythingOfType("*community.Group")).Return(nil)
	mockMembershipRepo.On("Save", mock.Anything, mock.AnythingOfType("*community.GroupMembership")).Return(nil)
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	// Act
	group, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, group)
	assert.True(t, group.Settings().HasMemberLimit())
	assert.Equal(t, 100, group.Settings().MaxMembers())
	assert.True(t, group.Settings().RequireApproval())

	mockGroupRepo.AssertExpectations(t)
	mockMembershipRepo.AssertExpectations(t)
}
