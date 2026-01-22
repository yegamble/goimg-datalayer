package testhelpers

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// MockGroupRepository is a mock implementation of community.GroupRepository.
type MockGroupRepository struct {
	mock.Mock
}

func (m *MockGroupRepository) FindByID(ctx context.Context, id community.GroupID) (*community.Group, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.Group), args.Error(1)
}

func (m *MockGroupRepository) FindBySlug(ctx context.Context, slug community.GroupSlug) (*community.Group, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.Group), args.Error(1)
}

func (m *MockGroupRepository) FindPublicGroups(ctx context.Context, filter community.GroupFilter, pagination shared.Pagination) ([]*community.Group, int, error) {
	args := m.Called(ctx, filter, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.Group), args.Int(1), args.Error(2)
}

func (m *MockGroupRepository) SearchGroups(ctx context.Context, query string, pagination shared.Pagination) ([]*community.Group, int, error) {
	args := m.Called(ctx, query, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.Group), args.Int(1), args.Error(2)
}

func (m *MockGroupRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*community.Group, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.Group), args.Int(1), args.Error(2)
}

func (m *MockGroupRepository) Save(ctx context.Context, group *community.Group) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockGroupRepository) Delete(ctx context.Context, id community.GroupID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGroupRepository) ExistsWithSlug(ctx context.Context, slug community.GroupSlug) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

// MockGroupMembershipRepository is a mock implementation of community.GroupMembershipRepository.
type MockGroupMembershipRepository struct {
	mock.Mock
}

func (m *MockGroupMembershipRepository) FindByID(ctx context.Context, id community.MembershipID) (*community.GroupMembership, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupMembership), args.Error(1)
}

func (m *MockGroupMembershipRepository) FindByGroupAndUser(ctx context.Context, groupID community.GroupID, userID identity.UserID) (*community.GroupMembership, error) {
	args := m.Called(ctx, groupID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupMembership), args.Error(1)
}

func (m *MockGroupMembershipRepository) FindByGroup(ctx context.Context, groupID community.GroupID, filter community.MemberFilter, pagination shared.Pagination) ([]*community.GroupMembership, int, error) {
	args := m.Called(ctx, groupID, filter, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupMembership), args.Int(1), args.Error(2)
}

func (m *MockGroupMembershipRepository) FindByUser(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]*community.GroupMembership, int, error) {
	args := m.Called(ctx, userID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupMembership), args.Int(1), args.Error(2)
}

func (m *MockGroupMembershipRepository) FindActiveByGroup(ctx context.Context, groupID community.GroupID) ([]*community.GroupMembership, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*community.GroupMembership), args.Error(1)
}

func (m *MockGroupMembershipRepository) FindPendingByGroup(ctx context.Context, groupID community.GroupID) ([]*community.GroupMembership, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*community.GroupMembership), args.Error(1)
}

func (m *MockGroupMembershipRepository) CountByGroupAndStatus(ctx context.Context, groupID community.GroupID, status community.MemberStatus) (int, error) {
	args := m.Called(ctx, groupID, status)
	return args.Int(0), args.Error(1)
}

func (m *MockGroupMembershipRepository) Save(ctx context.Context, membership *community.GroupMembership) error {
	args := m.Called(ctx, membership)
	return args.Error(0)
}

func (m *MockGroupMembershipRepository) Delete(ctx context.Context, id community.MembershipID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGroupMembershipRepository) ExistsActiveByGroupAndUser(ctx context.Context, groupID community.GroupID, userID identity.UserID) (bool, error) {
	args := m.Called(ctx, groupID, userID)
	return args.Bool(0), args.Error(1)
}

// MockEventPublisher is a mock implementation of EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockGroupImageRepository is a mock implementation of community.GroupImageRepository.
type MockGroupImageRepository struct {
	mock.Mock
}

func (m *MockGroupImageRepository) Save(ctx context.Context, groupImage *community.GroupImage) error {
	args := m.Called(ctx, groupImage)
	return args.Error(0)
}

func (m *MockGroupImageRepository) FindByID(ctx context.Context, id community.GroupImageID) (*community.GroupImage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupImage), args.Error(1)
}

func (m *MockGroupImageRepository) FindByGroupAndImage(ctx context.Context, groupID community.GroupID, imageID gallery.ImageID) (*community.GroupImage, error) {
	args := m.Called(ctx, groupID, imageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupImage), args.Error(1)
}

func (m *MockGroupImageRepository) FindByGroup(ctx context.Context, groupID community.GroupID, status *community.GroupImageStatus, pagination shared.Pagination) ([]*community.GroupImage, int, error) {
	args := m.Called(ctx, groupID, status, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupImage), args.Int(1), args.Error(2)
}

func (m *MockGroupImageRepository) FindPendingByGroup(ctx context.Context, groupID community.GroupID, pagination shared.Pagination) ([]*community.GroupImage, int, error) {
	args := m.Called(ctx, groupID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupImage), args.Int(1), args.Error(2)
}

func (m *MockGroupImageRepository) FindApprovedByGroup(ctx context.Context, groupID community.GroupID, pagination shared.Pagination) ([]*community.GroupImage, int, error) {
	args := m.Called(ctx, groupID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupImage), args.Int(1), args.Error(2)
}

func (m *MockGroupImageRepository) Delete(ctx context.Context, id community.GroupImageID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGroupImageRepository) ExistsByGroupAndImage(ctx context.Context, groupID community.GroupID, imageID gallery.ImageID) (bool, error) {
	args := m.Called(ctx, groupID, imageID)
	return args.Bool(0), args.Error(1)
}

// MockGroupInvitationRepository is a mock implementation of community.GroupInvitationRepository.
type MockGroupInvitationRepository struct {
	mock.Mock
}

func (m *MockGroupInvitationRepository) Save(ctx context.Context, invitation *community.GroupInvitation) error {
	args := m.Called(ctx, invitation)
	return args.Error(0)
}

func (m *MockGroupInvitationRepository) FindByID(ctx context.Context, id community.InvitationID) (*community.GroupInvitation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupInvitation), args.Error(1)
}

func (m *MockGroupInvitationRepository) FindByToken(ctx context.Context, token community.InvitationToken) (*community.GroupInvitation, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupInvitation), args.Error(1)
}

func (m *MockGroupInvitationRepository) FindPendingByGroup(ctx context.Context, groupID community.GroupID) ([]*community.GroupInvitation, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*community.GroupInvitation), args.Error(1)
}

func (m *MockGroupInvitationRepository) FindPendingByUser(ctx context.Context, userID identity.UserID) ([]*community.GroupInvitation, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*community.GroupInvitation), args.Error(1)
}

func (m *MockGroupInvitationRepository) Delete(ctx context.Context, id community.InvitationID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockGroupAlbumRepository is a mock implementation of community.GroupAlbumRepository.
type MockGroupAlbumRepository struct {
	mock.Mock
}

func (m *MockGroupAlbumRepository) Save(ctx context.Context, album *community.GroupAlbum) error {
	args := m.Called(ctx, album)
	return args.Error(0)
}

func (m *MockGroupAlbumRepository) FindByID(ctx context.Context, id community.GroupAlbumID) (*community.GroupAlbum, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*community.GroupAlbum), args.Error(1)
}

func (m *MockGroupAlbumRepository) FindByGroup(ctx context.Context, groupID community.GroupID, pagination shared.Pagination) ([]*community.GroupAlbum, int, error) {
	args := m.Called(ctx, groupID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupAlbum), args.Int(1), args.Error(2)
}

func (m *MockGroupAlbumRepository) FindByGroupAndCreator(ctx context.Context, groupID community.GroupID, creatorID identity.UserID, pagination shared.Pagination) ([]*community.GroupAlbum, int, error) {
	args := m.Called(ctx, groupID, creatorID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*community.GroupAlbum), args.Int(1), args.Error(2)
}

func (m *MockGroupAlbumRepository) Delete(ctx context.Context, id community.GroupAlbumID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockGroupAlbumImageRepository is a mock implementation of community.GroupAlbumImageRepository.
type MockGroupAlbumImageRepository struct {
	mock.Mock
}

func (m *MockGroupAlbumImageRepository) AddImageToAlbum(ctx context.Context, albumID community.GroupAlbumID, imageID gallery.ImageID, addedBy identity.UserID) error {
	args := m.Called(ctx, albumID, imageID, addedBy)
	return args.Error(0)
}

func (m *MockGroupAlbumImageRepository) RemoveImageFromAlbum(ctx context.Context, albumID community.GroupAlbumID, imageID gallery.ImageID) error {
	args := m.Called(ctx, albumID, imageID)
	return args.Error(0)
}

func (m *MockGroupAlbumImageRepository) IsImageInAlbum(ctx context.Context, albumID community.GroupAlbumID, imageID gallery.ImageID) (bool, error) {
	args := m.Called(ctx, albumID, imageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockGroupAlbumImageRepository) FindImagesInAlbum(ctx context.Context, albumID community.GroupAlbumID, pagination shared.Pagination) ([]*gallery.Image, int, error) {
	args := m.Called(ctx, albumID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*gallery.Image), args.Int(1), args.Error(2)
}

func (m *MockGroupAlbumImageRepository) GetImageAddedBy(ctx context.Context, albumID community.GroupAlbumID, imageID gallery.ImageID) (identity.UserID, error) {
	args := m.Called(ctx, albumID, imageID)
	return args.Get(0).(identity.UserID), args.Error(1)
}
