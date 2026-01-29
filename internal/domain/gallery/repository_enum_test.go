package gallery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestRepositoryEnums_Explicit(t *testing.T) {
	t.Parallel()

	t.Run("NSFWFilter", func(t *testing.T) {
		assert.True(t, gallery.NSFWFilterExcludeAll.IsValid())
		assert.True(t, gallery.NSFWFilterSafeOnly.IsValid())
		assert.True(t, gallery.NSFWFilterIncludeAll.IsValid())
		assert.True(t, gallery.NSFWFilterSuggestiveOK.IsValid())
		assert.False(t, gallery.NSFWFilter("invalid").IsValid())
	})

	t.Run("TagPeriod", func(t *testing.T) {
		assert.True(t, gallery.TagPeriodDay.IsValid())
		assert.True(t, gallery.TagPeriodWeek.IsValid())
		assert.True(t, gallery.TagPeriodMonth.IsValid())
		assert.True(t, gallery.TagPeriodAll.IsValid())
		assert.False(t, gallery.TagPeriod("invalid").IsValid())
	})
}

func TestRepositoryStructs(t *testing.T) {
	t.Parallel()

	t.Run("SearchParams", func(t *testing.T) {
		ownerID := identity.NewUserID()
		vis := gallery.VisibilityPublic
		nsfw := gallery.NSFWFilterExcludeAll

		params := gallery.SearchParams{
			Query:      "test",
			Tags:       []gallery.Tag{},
			OwnerID:    &ownerID,
			Visibility: &vis,
			NSFWFilter: &nsfw,
			SortBy:     gallery.SearchSortByRelevance,
		}
		assert.Equal(t, "test", params.Query)
	})

	t.Run("TagWithUsage", func(t *testing.T) {
		tag, _ := gallery.NewTag("test")
		tu := gallery.TagWithUsage{
			Tag:        tag,
			UsageCount: 10,
			TrendScore: 5.5,
		}
		assert.Equal(t, int64(10), tu.UsageCount)
		assert.InDelta(t, 5.5, tu.TrendScore, 0.001)
	})
}
