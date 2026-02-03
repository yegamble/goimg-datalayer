//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
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

func BenchmarkUserRepository_Save_Insert(b *testing.B) {
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

	repo := postgres.NewUserRepository(pg.DB)
	hash := getPrecomputedHash(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		email, _ := identity.NewEmail(fmt.Sprintf("bench_%d@example.com", i))
		username, _ := identity.NewUsername(fmt.Sprintf("bench_%d", i))
		user, _ := identity.NewUser(email, username, hash)
		user.ClearEvents()

		b.StartTimer()

		err := repo.Save(ctx, user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUserRepository_Save_Update(b *testing.B) {
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

	repo := postgres.NewUserRepository(pg.DB)
	hash := getPrecomputedHash(b)

	// Create a user to update
	email, _ := identity.NewEmail("bench_update@example.com")
	username, _ := identity.NewUsername("bench_update")
	user, _ := identity.NewUser(email, username, hash)
	user.ClearEvents()

	err = repo.Save(ctx, user)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just save the same user again (update)
		b.StopTimer()
		// We could modify something, but Save() updates everything anyway.
		// Let's keep it simple.
		b.StartTimer()

		err := repo.Save(ctx, user)
		if err != nil {
			b.Fatal(err)
		}
	}
}
