package testhelpers

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
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
