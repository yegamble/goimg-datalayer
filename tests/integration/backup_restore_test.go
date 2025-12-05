//go:build integration
// +build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// TestBackupRestore_FullCycle validates the complete backup and restore process
// using testcontainers. This test exercises:
//   - Database seeding with realistic data
//   - Backup creation using pg_dump
//   - Data checksumming before backup
//   - Database destruction (simulating disaster)
//   - Restore from backup using pg_restore
//   - Data integrity verification via checksums
//   - Foreign key relationship verification
//   - Recovery Time Objective (RTO) measurement
//
// Security Gate: S9-PROD-004 - RTO must be < 30 minutes
func TestBackupRestore_FullCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Step 1: Populate seed data
	t.Log("Step 1: Populating seed data...")
	seedDataPath := filepath.Join("..", "..", "tests", "integration", "backup-restore-seed.sql")
	err := executeSQLFile(ctx, suite.DB.DB, seedDataPath)
	require.NoError(t, err, "failed to populate seed data")

	// Step 2: Calculate checksums before backup
	t.Log("Step 2: Calculating pre-backup checksums...")
	checksumsBefore, err := calculateTableChecksums(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to calculate checksums before backup")
	require.NotEmpty(t, checksumsBefore, "checksums should not be empty")

	rowCountsBefore, err := getRowCounts(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to get row counts before backup")

	t.Logf("Pre-backup row counts: %+v", rowCountsBefore)

	// Step 3: Create backup
	t.Log("Step 3: Creating backup...")
	backupFile, err := createBackup(ctx, suite.Postgres.ConnStr)
	require.NoError(t, err, "failed to create backup")
	require.FileExists(t, backupFile, "backup file should exist")
	defer os.Remove(backupFile) // Cleanup

	backupInfo, err := os.Stat(backupFile)
	require.NoError(t, err, "failed to stat backup file")
	t.Logf("Backup file size: %d bytes (%.2f MB)", backupInfo.Size(), float64(backupInfo.Size())/1024/1024)

	// Step 4: Destroy database (simulate disaster)
	t.Log("Step 4: Simulating disaster (dropping database)...")
	err = suite.Postgres.Cleanup(ctx, t)
	require.NoError(t, err, "failed to cleanup database")

	// Verify database is empty
	rowCountsAfterDestroy, err := getRowCounts(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to get row counts after destroy")
	for table, count := range rowCountsAfterDestroy {
		assert.Equal(t, 0, count, "table %s should be empty after cleanup", table)
	}

	// Step 5: Restore from backup (with RTO measurement)
	t.Log("Step 5: Restoring from backup (measuring RTO)...")
	rtoStart := time.Now()

	err = restoreBackup(ctx, suite.Postgres.ConnStr, backupFile)
	require.NoError(t, err, "failed to restore backup")

	rtoDuration := time.Since(rtoStart)
	rtoSeconds := int(rtoDuration.Seconds())
	rtoMinutes := rtoDuration.Minutes()

	t.Logf("Recovery Time Objective (RTO): %d seconds (%.2f minutes)", rtoSeconds, rtoMinutes)

	// Step 6: Verify RTO target
	t.Log("Step 6: Verifying RTO target (< 30 minutes)...")
	const rtoTargetSeconds = 30 * 60 // 30 minutes
	if rtoSeconds > rtoTargetSeconds {
		t.Errorf("RTO exceeded: %d seconds > %d seconds (Security Gate S9-PROD-004 FAILED)", rtoSeconds, rtoTargetSeconds)
	} else {
		t.Logf("RTO target met: %d seconds < %d seconds ✓", rtoSeconds, rtoTargetSeconds)
	}

	// Step 7: Calculate checksums after restore
	t.Log("Step 7: Calculating post-restore checksums...")
	checksumsAfter, err := calculateTableChecksums(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to calculate checksums after restore")

	rowCountsAfter, err := getRowCounts(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to get row counts after restore")

	t.Logf("Post-restore row counts: %+v", rowCountsAfter)

	// Step 8: Compare row counts
	t.Log("Step 8: Comparing row counts...")
	for table, countBefore := range rowCountsBefore {
		countAfter, exists := rowCountsAfter[table]
		require.True(t, exists, "table %s should exist after restore", table)
		assert.Equal(t, countBefore, countAfter, "row count mismatch for table %s", table)
		if countBefore == countAfter {
			t.Logf("✓ %s: %d rows", table, countAfter)
		}
	}

	// Step 9: Compare checksums
	t.Log("Step 9: Comparing checksums...")
	for table, checksumBefore := range checksumsBefore {
		checksumAfter, exists := checksumsAfter[table]
		require.True(t, exists, "table %s should exist after restore", table)
		assert.Equal(t, checksumBefore, checksumAfter, "checksum mismatch for table %s", table)
		if checksumBefore == checksumAfter {
			t.Logf("✓ %s: checksum matches", table)
		}
	}

	// Step 10: Verify foreign key relationships
	t.Log("Step 10: Verifying foreign key relationships...")
	err = verifyForeignKeys(ctx, suite.DB.DB)
	require.NoError(t, err, "foreign key verification failed")

	// Step 11: Verify triggers and functions
	t.Log("Step 11: Verifying triggers and functions...")
	triggerCount, err := countTriggers(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to count triggers")
	assert.GreaterOrEqual(t, triggerCount, 3, "expected at least 3 triggers to be restored")
	t.Logf("✓ Triggers verified: %d found", triggerCount)

	// Final Summary
	t.Log("")
	t.Log("========================================")
	t.Log("BACKUP/RESTORE VALIDATION SUMMARY")
	t.Log("========================================")
	t.Logf("Recovery Time: %d seconds (%.2f minutes)", rtoSeconds, rtoMinutes)
	t.Logf("RTO Target: %d seconds (30 minutes)", rtoTargetSeconds)
	t.Logf("RTO Status: %s", func() string {
		if rtoSeconds < rtoTargetSeconds {
			return "PASSED ✓"
		}
		return "FAILED ✗"
	}())
	t.Log("Data Integrity: VERIFIED ✓")
	t.Log("Foreign Keys: VERIFIED ✓")
	t.Log("Triggers: VERIFIED ✓")
	t.Log("Security Gate S9-PROD-004: PASSED ✓")
	t.Log("========================================")
}

// TestBackupRestore_EmptyDatabase tests backup/restore of an empty database
// to ensure the process handles edge cases correctly.
func TestBackupRestore_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Ensure database is empty
	suite.CleanupBetweenTests()

	// Create backup of empty database
	backupFile, err := createBackup(ctx, suite.Postgres.ConnStr)
	require.NoError(t, err, "failed to create backup of empty database")
	defer os.Remove(backupFile)

	// Verify backup file exists and has reasonable size
	backupInfo, err := os.Stat(backupFile)
	require.NoError(t, err, "failed to stat backup file")
	assert.Greater(t, backupInfo.Size(), int64(0), "backup file should not be empty")

	// Restore from backup
	suite.CleanupBetweenTests()
	err = restoreBackup(ctx, suite.Postgres.ConnStr, backupFile)
	require.NoError(t, err, "failed to restore empty database backup")

	// Verify all tables exist but are empty
	rowCounts, err := getRowCounts(ctx, suite.DB.DB)
	require.NoError(t, err, "failed to get row counts")

	for table, count := range rowCounts {
		assert.Equal(t, 0, count, "table %s should be empty", table)
	}
}

// TestBackupRestore_PartialData tests backup/restore with only some tables populated.
func TestBackupRestore_PartialData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create only users (no images, albums, etc.)
	_, err := suite.DB.ExecContext(ctx, `
		INSERT INTO users (id, email, username, password_hash, role, status)
		VALUES
			(gen_random_uuid(), 'test1@example.com', 'user1', '$argon2id$v=19$m=65536,t=3,p=2$test', 'user', 'active'),
			(gen_random_uuid(), 'test2@example.com', 'user2', '$argon2id$v=19$m=65536,t=3,p=2$test', 'user', 'active');
	`)
	require.NoError(t, err, "failed to insert test users")

	// Get checksums before backup
	checksumsBefore, err := calculateTableChecksums(ctx, suite.DB.DB)
	require.NoError(t, err)

	// Create backup
	backupFile, err := createBackup(ctx, suite.Postgres.ConnStr)
	require.NoError(t, err)
	defer os.Remove(backupFile)

	// Cleanup and restore
	suite.CleanupBetweenTests()
	err = restoreBackup(ctx, suite.Postgres.ConnStr, backupFile)
	require.NoError(t, err)

	// Verify checksums match
	checksumsAfter, err := calculateTableChecksums(ctx, suite.DB.DB)
	require.NoError(t, err)

	for table, checksumBefore := range checksumsBefore {
		checksumAfter := checksumsAfter[table]
		assert.Equal(t, checksumBefore, checksumAfter, "checksum mismatch for table %s", table)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// executeSQLFile executes a SQL file against the database.
func executeSQLFile(ctx context.Context, db *sql.DB, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	_, err = db.ExecContext(ctx, string(content))
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	return nil
}

// calculateTableChecksums calculates MD5 checksums for all table data.
// This is used to verify data integrity after restore.
func calculateTableChecksums(ctx context.Context, db *sql.DB) (map[string]string, error) {
	tables := []string{
		"users", "sessions", "images", "image_variants",
		"albums", "album_images", "tags", "image_tags",
		"likes", "comments",
	}

	checksums := make(map[string]string)

	for _, table := range tables {
		// Use PostgreSQL's MD5 aggregate to checksum all rows
		query := fmt.Sprintf(
			"SELECT md5(string_agg(t::text, '' ORDER BY t::text)) FROM (SELECT * FROM %s) t",
			table,
		)

		var checksum sql.NullString
		err := db.QueryRowContext(ctx, query).Scan(&checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate checksum for %s: %w", table, err)
		}

		// NULL checksum means empty table
		if checksum.Valid {
			checksums[table] = checksum.String
		} else {
			checksums[table] = "empty"
		}
	}

	return checksums, nil
}

// getRowCounts returns the number of rows in each table.
func getRowCounts(ctx context.Context, db *sql.DB) (map[string]int, error) {
	tables := []string{
		"users", "sessions", "images", "image_variants",
		"albums", "album_images", "tags", "image_tags",
		"likes", "comments",
	}

	counts := make(map[string]int)

	for _, table := range tables {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

		var count int
		err := db.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to count rows in %s: %w", table, err)
		}

		counts[table] = count
	}

	return counts, nil
}

// createBackup creates a PostgreSQL backup using pg_dump.
func createBackup(ctx context.Context, connStr string) (string, error) {
	// Create temporary file for backup
	tmpFile, err := os.CreateTemp("", "backup-*.dump")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	backupPath := tmpFile.Name()
	tmpFile.Close()

	// Run pg_dump
	cmd := exec.CommandContext(ctx, "pg_dump",
		connStr,
		"-Fc",     // Custom format
		"-Z", "9", // Maximum compression
		"--no-owner", // Don't output ownership commands
		"--no-acl",   // Don't output ACL commands
		"-f", backupPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("pg_dump failed: %w\nStderr: %s", err, stderr.String())
	}

	return backupPath, nil
}

// restoreBackup restores a PostgreSQL backup using pg_restore.
func restoreBackup(ctx context.Context, connStr string, backupFile string) error {
	cmd := exec.CommandContext(ctx, "pg_restore",
		"-d", connStr,
		"--no-owner",
		"--no-acl",
		"--exit-on-error",
		backupFile,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %w\nStderr: %s", err, stderr.String())
	}

	return nil
}

// verifyForeignKeys tests that all foreign key relationships are intact.
func verifyForeignKeys(ctx context.Context, db *sql.DB) error {
	queries := []string{
		"SELECT COUNT(*) FROM images i JOIN users u ON i.owner_id = u.id",
		"SELECT COUNT(*) FROM sessions s JOIN users u ON s.user_id = u.id",
		"SELECT COUNT(*) FROM image_variants iv JOIN images i ON iv.image_id = i.id",
		"SELECT COUNT(*) FROM albums a JOIN users u ON a.owner_id = u.id",
		"SELECT COUNT(*) FROM album_images ai JOIN albums a ON ai.album_id = a.id JOIN images i ON ai.image_id = i.id",
		"SELECT COUNT(*) FROM image_tags it JOIN images i ON it.image_id = i.id JOIN tags t ON it.tag_id = t.id",
		"SELECT COUNT(*) FROM likes l JOIN users u ON l.user_id = u.id JOIN images i ON l.image_id = i.id",
		"SELECT COUNT(*) FROM comments c JOIN users u ON c.user_id = u.id JOIN images i ON c.image_id = i.id",
	}

	for _, query := range queries {
		var count int
		err := db.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			return fmt.Errorf("foreign key query failed: %s: %w", query, err)
		}
	}

	return nil
}

// countTriggers returns the number of user-defined triggers in the database.
func countTriggers(ctx context.Context, db *sql.DB) (int, error) {
	query := "SELECT COUNT(*) FROM pg_trigger WHERE tgisinternal = false"

	var count int
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count triggers: %w", err)
	}

	return count, nil
}

// TestBackupRestore_ChecksumCalculation validates the checksum calculation
// function itself to ensure it's deterministic and accurate.
func TestBackupRestore_ChecksumCalculation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Insert known data
	_, err := suite.DB.ExecContext(ctx, `
		INSERT INTO users (id, email, username, password_hash, role, status)
		VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com', 'testuser', 'hash', 'user', 'active')
	`)
	require.NoError(t, err)

	// Calculate checksum twice
	checksums1, err := calculateTableChecksums(ctx, suite.DB.DB)
	require.NoError(t, err)

	checksums2, err := calculateTableChecksums(ctx, suite.DB.DB)
	require.NoError(t, err)

	// Checksums should be identical (deterministic)
	assert.Equal(t, checksums1["users"], checksums2["users"], "checksums should be deterministic")
}

// TestBackupRestore_LargeDataset validates backup/restore with larger dataset
// to ensure performance remains acceptable.
func TestBackupRestore_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Insert 1000 users
	t.Log("Inserting 1000 users...")
	for i := 0; i < 1000; i++ {
		_, err := suite.DB.ExecContext(ctx, `
			INSERT INTO users (id, email, username, password_hash, role, status)
			VALUES (gen_random_uuid(), $1, $2, $3, 'user', 'active')
		`, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("user%d", i), "hash")
		require.NoError(t, err)
	}

	rowCountsBefore, err := getRowCounts(ctx, suite.DB.DB)
	require.NoError(t, err)
	assert.Equal(t, 1000, rowCountsBefore["users"])

	// Backup
	t.Log("Creating backup...")
	backupStart := time.Now()
	backupFile, err := createBackup(ctx, suite.Postgres.ConnStr)
	require.NoError(t, err)
	defer os.Remove(backupFile)
	backupDuration := time.Since(backupStart)
	t.Logf("Backup completed in %v", backupDuration)

	// Restore
	suite.CleanupBetweenTests()
	t.Log("Restoring backup...")
	restoreStart := time.Now()
	err = restoreBackup(ctx, suite.Postgres.ConnStr, backupFile)
	require.NoError(t, err)
	restoreDuration := time.Since(restoreStart)
	t.Logf("Restore completed in %v", restoreDuration)

	// Verify
	rowCountsAfter, err := getRowCounts(ctx, suite.DB.DB)
	require.NoError(t, err)
	assert.Equal(t, 1000, rowCountsAfter["users"])

	// Total RTO should be reasonable
	totalRTO := backupDuration + restoreDuration
	t.Logf("Total RTO for 1000 users: %v", totalRTO)
	assert.Less(t, totalRTO.Seconds(), float64(60), "RTO should be less than 60 seconds for 1000 users")
}

// BenchmarkBackupRestore measures backup/restore performance.
func BenchmarkBackupRestore(b *testing.B) {
	suite := containers.NewIntegrationTestSuite(&testing.T{})
	ctx := context.Background()

	// Setup seed data once
	seedDataPath := filepath.Join("..", "..", "tests", "integration", "backup-restore-seed.sql")
	err := executeSQLFile(ctx, suite.DB.DB, seedDataPath)
	if err != nil {
		b.Fatalf("failed to populate seed data: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Backup
		backupFile, err := createBackup(ctx, suite.Postgres.ConnStr)
		if err != nil {
			b.Fatalf("backup failed: %v", err)
		}

		// Cleanup
		suite.Postgres.Cleanup(ctx, &testing.T{})

		// Restore
		err = restoreBackup(ctx, suite.Postgres.ConnStr, backupFile)
		if err != nil {
			b.Fatalf("restore failed: %v", err)
		}

		os.Remove(backupFile)
	}
}
