package community_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestRepositoryStructs(t *testing.T) {
	t.Parallel()

	t.Run("GroupFilter", func(t *testing.T) {
		gt := community.GroupTypePublic
		ownerID := identity.NewUserID()

		filter := community.GroupFilter{
			GroupType: &gt,
			OwnerID:   &ownerID,
			SortBy:    community.GroupSortByPopular,
		}
		assert.Equal(t, community.GroupSortByPopular, filter.SortBy)
	})

	t.Run("MemberFilter", func(t *testing.T) {
		role := community.GroupRoleAdmin
		status := community.MemberStatusActive

		filter := community.MemberFilter{
			Role:   &role,
			Status: &status,
		}
		assert.Equal(t, &role, filter.Role)
	})
}
