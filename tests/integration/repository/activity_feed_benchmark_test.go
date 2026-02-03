//go:build integration
// +build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

func getPrecomputedHash(t testing.TB) identity.PasswordHash {
	h, err := identity.NewPasswordHash("Password123!")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func BenchmarkActivityRepository_FindFeedForUser(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmark in short mode")
	}
	ctx := context.Background()

	// Setup Postgres container manually
	pg, err := containers.NewPostgresContainer(ctx, b)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		pg.Terminate(ctx)
	})

	userRepo := postgres.NewUserRepository(pg.DB)
	activityRepo := postgres.NewActivityRepository(pg.DB)
	followRepo := postgres.NewFollowRepository(pg.DB)

	hash := getPrecomputedHash(b)

	// 1. Create Follower User
	followerEmail, _ := identity.NewEmail("follower@example.com")
	followerUsername, _ := identity.NewUsername("follower")
	follower, _ := identity.NewUser(followerEmail, followerUsername, hash)
	follower.ClearEvents()
	err = userRepo.Save(ctx, follower)
	require.NoError(b, err)

	// 2. Create Followed Users and Activities
	const numFollowed = 50
	const activitiesPerUser = 100

	for i := 0; i < numFollowed; i++ {
		// Create Followed User
		followedEmail, _ := identity.NewEmail(fmt.Sprintf("followed_%d@example.com", i))
		followedUsername, _ := identity.NewUsername(fmt.Sprintf("followed_%d", i))
		followed, _ := identity.NewUser(followedEmail, followedUsername, hash)
		followed.ClearEvents()
		err = userRepo.Save(ctx, followed)
		require.NoError(b, err)

		// Follow
		follow, err := identity.NewFollow(follower.ID(), followed.ID())
		require.NoError(b, err)
		err = followRepo.Save(ctx, follow)
		require.NoError(b, err)

		// Create Activities
		for j := 0; j < activitiesPerUser; j++ {
			actType, _ := activity.ParseActivityType("image_uploaded")
			targetType, _ := activity.ParseTargetType("image")
			targetID := uuid.New()

			// Distribute creation times to make sorting interesting
			// Create activities over the last 30 days
			createdAt := time.Now().Add(-time.Duration(j) * time.Hour).Add(-time.Duration(i) * time.Second)

			act := activity.ReconstructActivity(
				activity.NewActivityID(),
				followed.ID(),
				actType,
				targetID,
				targetType,
				map[string]string{"foo": "bar"},
				createdAt,
			)
			err = activityRepo.Save(ctx, act)
			require.NoError(b, err)
		}
	}

	pagination, _ := shared.NewPagination(1, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := activityRepo.FindFeedForUser(ctx, follower.ID(), pagination)
		if err != nil {
			b.Fatal(err)
		}
	}
}
