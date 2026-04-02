package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// SQL queries for device operations.
const (
	sqlUpsertDevice = `
		INSERT INTO user_devices (id, user_id, fingerprint_hash, ip_address, user_agent, device_name, trusted, first_seen_at, last_seen_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, fingerprint_hash)
		DO UPDATE SET
		    ip_address = EXCLUDED.ip_address,
		    user_agent = EXCLUDED.user_agent,
		    device_name = EXCLUDED.device_name,
		    last_seen_at = EXCLUDED.last_seen_at
		RETURNING id, first_seen_at
	`

	sqlSelectDevicesByUserID = `
		SELECT id, user_id, fingerprint_hash, ip_address, user_agent, device_name, trusted, first_seen_at, last_seen_at, created_at
		FROM user_devices
		WHERE user_id = $1
		ORDER BY last_seen_at DESC
	`

	sqlSelectDeviceByFingerprint = `
		SELECT id, user_id, fingerprint_hash, ip_address, user_agent, device_name, trusted, first_seen_at, last_seen_at, created_at
		FROM user_devices
		WHERE user_id = $1 AND fingerprint_hash = $2
	`

	sqlSelectTrustedDevices = `
		SELECT id, user_id, fingerprint_hash, ip_address, user_agent, device_name, trusted, first_seen_at, last_seen_at, created_at
		FROM user_devices
		WHERE user_id = $1 AND trusted = true
		ORDER BY last_seen_at DESC
	`

	sqlUpdateDeviceTrusted = `
		UPDATE user_devices
		SET trusted = $2
		WHERE id = $1
	`

	sqlDeleteDevice = `
		DELETE FROM user_devices
		WHERE id = $1 AND user_id = $2
	`

	sqlDeleteDevicesByUserID = `
		DELETE FROM user_devices
		WHERE user_id = $1
	`

	sqlCountDevicesByUserID = `
		SELECT COUNT(*)
		FROM user_devices
		WHERE user_id = $1
	`
)

// deviceRow represents a device row in the database.
type deviceRow struct {
	ID              string    `db:"id"`
	UserID          string    `db:"user_id"`
	FingerprintHash string    `db:"fingerprint_hash"`
	IPAddress       string    `db:"ip_address"`
	UserAgent       string    `db:"user_agent"`
	DeviceName      string    `db:"device_name"`
	Trusted         bool      `db:"trusted"`
	FirstSeenAt     time.Time `db:"first_seen_at"`
	LastSeenAt      time.Time `db:"last_seen_at"`
	CreatedAt       time.Time `db:"created_at"`
}

// DeviceRepository handles persistence of user devices.
type DeviceRepository struct {
	db *sqlx.DB
}

// NewDeviceRepository creates a new device repository.
func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// TrackDeviceResult contains the result of tracking a device.
type TrackDeviceResult struct {
	DeviceID    string
	IsNewDevice bool
	FirstSeenAt time.Time
}

// Track records a device login and returns whether it's a new device.
// Uses upsert to update last_seen_at if device exists.
func (r *DeviceRepository) Track(ctx context.Context, userID identity.UserID, device identity.DeviceFingerprint) (*TrackDeviceResult, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	var resultID string
	var firstSeenAt time.Time

	err := r.db.QueryRowContext(
		ctx,
		sqlUpsertDevice,
		id,
		userID.String(),
		device.FingerprintHash(),
		device.IPAddress(),
		device.UserAgent(),
		device.DeviceName(),
		device.IsTrusted(),
		now, // first_seen_at (only used on insert)
		now, // last_seen_at
		now, // created_at
	).Scan(&resultID, &firstSeenAt)

	if err != nil {
		return nil, fmt.Errorf("failed to track device: %w", err)
	}

	// If the returned ID matches our generated one, it's a new device
	isNew := resultID == id

	return &TrackDeviceResult{
		DeviceID:    resultID,
		IsNewDevice: isNew,
		FirstSeenAt: firstSeenAt,
	}, nil
}

// FindByUserID retrieves all devices for a user.
func (r *DeviceRepository) FindByUserID(ctx context.Context, userID identity.UserID) ([]identity.DeviceFingerprint, error) {
	var rows []deviceRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectDevicesByUserID, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find devices: %w", err)
	}

	devices := make([]identity.DeviceFingerprint, len(rows))
	for i, row := range rows {
		devices[i] = identity.ReconstructDeviceFingerprint(
			row.FingerprintHash,
			row.IPAddress,
			row.UserAgent,
			row.DeviceName,
			row.Trusted,
			row.FirstSeenAt,
			row.LastSeenAt,
		)
	}

	return devices, nil
}

// FindByFingerprint retrieves a specific device by its fingerprint hash.
func (r *DeviceRepository) FindByFingerprint(ctx context.Context, userID identity.UserID, fingerprintHash string) (*identity.DeviceFingerprint, error) {
	var row deviceRow
	err := r.db.GetContext(ctx, &row, sqlSelectDeviceByFingerprint, userID.String(), fingerprintHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Device not found is not an error
		}
		return nil, fmt.Errorf("failed to find device: %w", err)
	}

	device := identity.ReconstructDeviceFingerprint(
		row.FingerprintHash,
		row.IPAddress,
		row.UserAgent,
		row.DeviceName,
		row.Trusted,
		row.FirstSeenAt,
		row.LastSeenAt,
	)

	return &device, nil
}

// FindTrustedDevices retrieves all trusted devices for a user.
func (r *DeviceRepository) FindTrustedDevices(ctx context.Context, userID identity.UserID) ([]identity.DeviceFingerprint, error) {
	var rows []deviceRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectTrustedDevices, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find trusted devices: %w", err)
	}

	devices := make([]identity.DeviceFingerprint, len(rows))
	for i, row := range rows {
		devices[i] = identity.ReconstructDeviceFingerprint(
			row.FingerprintHash,
			row.IPAddress,
			row.UserAgent,
			row.DeviceName,
			row.Trusted,
			row.FirstSeenAt,
			row.LastSeenAt,
		)
	}

	return devices, nil
}

// SetTrusted marks a device as trusted or untrusted.
func (r *DeviceRepository) SetTrusted(ctx context.Context, deviceID string, trusted bool) error {
	result, err := r.db.ExecContext(ctx, sqlUpdateDeviceTrusted, deviceID, trusted)
	if err != nil {
		return fmt.Errorf("failed to update device trust status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("device not found")
	}

	return nil
}

// SetTrustedByFingerprint marks a device as trusted by its fingerprint hash.
func (r *DeviceRepository) SetTrustedByFingerprint(ctx context.Context, userID identity.UserID, fingerprintHash string, trusted bool) error {
	var row deviceRow
	err := r.db.GetContext(ctx, &row, sqlSelectDeviceByFingerprint, userID.String(), fingerprintHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("device not found")
		}
		return fmt.Errorf("failed to find device: %w", err)
	}

	return r.SetTrusted(ctx, row.ID, trusted)
}

// Delete removes a specific device.
func (r *DeviceRepository) Delete(ctx context.Context, userID identity.UserID, deviceID string) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteDevice, deviceID, userID.String())
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("device not found")
	}

	return nil
}

// DeleteAllByUserID removes all devices for a user.
func (r *DeviceRepository) DeleteAllByUserID(ctx context.Context, userID identity.UserID) error {
	_, err := r.db.ExecContext(ctx, sqlDeleteDevicesByUserID, userID.String())
	if err != nil {
		return fmt.Errorf("failed to delete devices: %w", err)
	}
	return nil
}

// Count returns the number of devices for a user.
func (r *DeviceRepository) Count(ctx context.Context, userID identity.UserID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, sqlCountDevicesByUserID, userID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to count devices: %w", err)
	}
	return count, nil
}

// IsKnownDevice checks if a device fingerprint is known for a user.
func (r *DeviceRepository) IsKnownDevice(ctx context.Context, userID identity.UserID, fingerprintHash string) (bool, error) {
	device, err := r.FindByFingerprint(ctx, userID, fingerprintHash)
	if err != nil {
		return false, err
	}
	return device != nil, nil
}
