package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Guest User Tests

func TestNewGuestUser(t *testing.T) {
	t.Parallel()

	t.Run("creates valid guest user", func(t *testing.T) {
		t.Parallel()

		ipAddress := "192.168.1.1"
		user, err := NewGuestUser(ipAddress)

		require.NoError(t, err)
		assert.False(t, user.ID().IsZero())
		assert.Equal(t, UserTypeGuest, user.UserType())
		assert.Equal(t, RoleUser, user.Role())
		assert.Equal(t, StatusActive, user.Status())
		assert.True(t, user.IsGuest())
		assert.NotNil(t, user.IPAddress())
		assert.Equal(t, ipAddress, *user.IPAddress())
		assert.NotNil(t, user.ExpiresAt())
		assert.False(t, user.IsExpired())

		// Username should start with "guest_"
		assert.Contains(t, user.Username().String(), "guest_")

		// Should have an expiration time (30 days from now)
		expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
		assert.True(t, user.ExpiresAt().After(time.Now()))
		assert.True(t, user.ExpiresAt().Before(expectedExpiry.Add(1*time.Minute)))

		// Should emit guest created event
		events := user.Events()
		assert.Len(t, events, 1)
		assert.Equal(t, "identity.guest.created", events[0].EventType())
	})

	t.Run("empty IP address fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewGuestUser("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ip address is required")
	})
}

func TestUser_IsGuest(t *testing.T) {
	t.Parallel()

	t.Run("guest user returns true", func(t *testing.T) {
		t.Parallel()

		user, _ := NewGuestUser("192.168.1.1")
		assert.True(t, user.IsGuest())
	})

	t.Run("registered user returns false", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.False(t, user.IsGuest())
	})
}

func TestUser_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("guest with future expiry is not expired", func(t *testing.T) {
		t.Parallel()

		user, _ := NewGuestUser("192.168.1.1")
		assert.False(t, user.IsExpired())
	})

	t.Run("guest with past expiry is expired", func(t *testing.T) {
		t.Parallel()

		ipAddress := "192.168.1.1"
		pastExpiry := time.Now().Add(-1 * time.Hour)
		user := ReconstructUser(
			NewUserID(),
			Email{},
			Username{},
			PasswordHash{},
			RoleUser,
			StatusActive,
			"guest",
			"",
			time.Now(),
			time.Now(),
			UserTypeGuest,
			&ipAddress,
			&pastExpiry,
			0,
		)

		assert.True(t, user.IsExpired())
	})

	t.Run("registered user with nil expiry is not expired", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.False(t, user.IsExpired())
		assert.Nil(t, user.ExpiresAt())
	})
}

func TestUser_ConvertToRegistered(t *testing.T) {
	t.Parallel()

	t.Run("converts guest to registered user", func(t *testing.T) {
		t.Parallel()

		guest, _ := NewGuestUser("192.168.1.1")
		guest.ClearEvents()

		email, _ := NewEmail("converted@example.com")
		username, _ := NewUsername("converteduser")
		password, _ := NewPasswordHash("ValidPassword123!")

		err := guest.ConvertToRegistered(email, username, password)
		require.NoError(t, err)

		assert.False(t, guest.IsGuest())
		assert.Equal(t, UserTypeRegistered, guest.UserType())
		assert.Equal(t, email, guest.Email())
		assert.Equal(t, username, guest.Username())
		assert.Nil(t, guest.IPAddress())
		assert.Nil(t, guest.ExpiresAt())

		// Should emit conversion event
		events := guest.Events()
		assert.Len(t, events, 1)
		assert.Equal(t, "identity.guest.converted", events[0].EventType())
	})

	t.Run("converting registered user fails", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		newEmail, _ := NewEmail("new@example.com")
		newUsername, _ := NewUsername("newuser")
		newPassword, _ := NewPasswordHash("ValidPassword123!")

		err := user.ConvertToRegistered(newEmail, newUsername, newPassword)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user is not a guest account")
	})
}

// 2FA Tests

func TestUser_TOTPSecret(t *testing.T) {
	t.Parallel()

	secret, _ := NewTOTPSecret([]byte("encrypted-secret"), "user@example.com")
	user := ReconstructUserWith2FA(
		NewUserID(),
		Email{},
		Username{},
		PasswordHash{},
		RoleUser,
		StatusActive,
		"User",
		"",
		time.Now(),
		time.Now(),
		&secret,
		nil,
		nil,
		UserTypeRegistered,
		nil,
		nil,
		0,
	)

	assert.NotNil(t, user.TOTPSecret())
	assert.Equal(t, secret.EncryptedSecret(), user.TOTPSecret().EncryptedSecret())
}

func TestUser_IsTOTPEnabled(t *testing.T) {
	t.Parallel()

	t.Run("returns true when TOTP is enabled", func(t *testing.T) {
		t.Parallel()

		secret := ReconstructTOTPSecret(
			[]byte("secret"),
			"goimg",
			"user@example.com",
			true,
			time.Now(),
		)
		user := ReconstructUserWith2FA(
			NewUserID(), Email{}, Username{}, PasswordHash{},
			RoleUser, StatusActive, "User", "",
			time.Now(), time.Now(),
			&secret, nil, nil,
			UserTypeRegistered, nil, nil, 0,
		)

		assert.True(t, user.IsTOTPEnabled())
	})

	t.Run("returns false when TOTP is not enabled", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.False(t, user.IsTOTPEnabled())
	})
}

func TestUser_IsTOTPSetupPending(t *testing.T) {
	t.Parallel()

	t.Run("returns true when setup is pending", func(t *testing.T) {
		t.Parallel()

		secret, _ := NewTOTPSecret([]byte("secret"), "user@example.com")
		user := ReconstructUserWith2FA(
			NewUserID(), Email{}, Username{}, PasswordHash{},
			RoleUser, StatusActive, "User", "",
			time.Now(), time.Now(),
			&secret, nil, nil,
			UserTypeRegistered, nil, nil, 0,
		)

		assert.True(t, user.IsTOTPSetupPending())
	})

	t.Run("returns false when no TOTP setup", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.False(t, user.IsTOTPSetupPending())
	})
}

func TestUser_SetupTOTP(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("user@example.com")
	username, _ := NewUsername("testuser")
	password, _ := NewPasswordHash("ValidPassword123!")
	user, _ := NewUser(email, username, password)
	user.ClearEvents()

	encryptedSecret := []byte("encrypted-totp-secret")
	err := user.SetupTOTP(encryptedSecret)
	require.NoError(t, err)

	assert.NotNil(t, user.TOTPSecret())
	assert.True(t, user.IsTOTPSetupPending())
	assert.False(t, user.IsTOTPEnabled())
}

func TestUser_EnableTOTP(t *testing.T) {
	t.Parallel()

	t.Run("enables TOTP after setup", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		encryptedSecret := []byte("encrypted-secret")
		_ = user.SetupTOTP(encryptedSecret)
		user.ClearEvents()

		// Generate backup codes for 2FA
		_, backupCodes, _ := GenerateBackupCodes()

		err := user.EnableTOTP(backupCodes)
		require.NoError(t, err)

		assert.True(t, user.IsTOTPEnabled())
		assert.False(t, user.IsTOTPSetupPending())

		// Should emit TOTP enabled event
		events := user.Events()
		assert.Len(t, events, 1)
		assert.Equal(t, "identity.user.totp_enabled", events[0].EventType())
	})

	t.Run("fails when TOTP not set up", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		_, backupCodes, _ := GenerateBackupCodes()
		err := user.EnableTOTP(backupCodes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "two-factor authentication is not enabled")
	})
}

func TestUser_DisableTOTP(t *testing.T) {
	t.Parallel()

	// Setup user with enabled TOTP
	secret := ReconstructTOTPSecret(
		[]byte("secret"), "goimg", "user@example.com", true, time.Now(),
	)
	user := ReconstructUserWith2FA(
		NewUserID(), Email{}, Username{}, PasswordHash{},
		RoleUser, StatusActive, "User", "",
		time.Now(), time.Now(),
		&secret, nil, nil,
		UserTypeRegistered, nil, nil, 0,
	)

	user.ClearEvents()

	err := user.DisableTOTP()
	require.NoError(t, err)

	assert.False(t, user.IsTOTPEnabled())

	// Should emit TOTP disabled event
	events := user.Events()
	assert.Len(t, events, 1)
	assert.Equal(t, "identity.user.totp_disabled", events[0].EventType())
}

// Backup Code Tests

func TestUser_BackupCodes(t *testing.T) {
	t.Parallel()

	_, codes, _ := GenerateBackupCodes()
	user := ReconstructUserWith2FA(
		NewUserID(), Email{}, Username{}, PasswordHash{},
		RoleUser, StatusActive, "User", "",
		time.Now(), time.Now(),
		nil, codes, nil,
		UserTypeRegistered, nil, nil, 0,
	)

	assert.Len(t, user.BackupCodes(), 10)
}

func TestUser_RegenerateBackupCodes(t *testing.T) {
	t.Parallel()

	t.Run("successfully regenerates backup codes when TOTP enabled", func(t *testing.T) {
		t.Parallel()

		// Setup user with TOTP enabled
		secret := ReconstructTOTPSecret(
			[]byte("secret"), "goimg", "user@example.com", true, time.Now(),
		)
		user := ReconstructUserWith2FA(
			NewUserID(), Email{}, Username{}, PasswordHash{},
			RoleUser, StatusActive, "User", "",
			time.Now(), time.Now(),
			&secret, nil, nil,
			UserTypeRegistered, nil, nil, 0,
		)
		user.ClearEvents()

		// Generate new codes
		_, newCodes, _ := GenerateBackupCodes()

		err := user.RegenerateBackupCodes(newCodes)
		require.NoError(t, err)

		assert.Len(t, user.BackupCodes(), 10)

		// Should emit event
		events := user.Events()
		assert.Len(t, events, 1)
		assert.Equal(t, "identity.user.backup_codes_regenerated", events[0].EventType())
	})

	t.Run("fails when TOTP not enabled", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		_, newCodes, _ := GenerateBackupCodes()
		err := user.RegenerateBackupCodes(newCodes)
		require.Error(t, err)
	})
}

func TestUser_UseBackupCode(t *testing.T) {
	t.Parallel()

	t.Run("successfully uses valid backup code", func(t *testing.T) {
		t.Parallel()

		secret := ReconstructTOTPSecret(
			[]byte("secret"), "goimg", "user@example.com", true, time.Now(),
		)
		plaintext, codes, _ := GenerateBackupCodes()
		user := ReconstructUserWith2FA(
			NewUserID(), Email{}, Username{}, PasswordHash{},
			RoleUser, StatusActive, "User", "",
			time.Now(), time.Now(),
			&secret, codes, nil,
			UserTypeRegistered, nil, nil, 0,
		)
		user.ClearEvents()

		err := user.UseBackupCode(plaintext[0])
		require.NoError(t, err)

		// Should emit event
		events := user.Events()
		assert.Len(t, events, 1)
		assert.Equal(t, "identity.user.backup_code_used", events[0].EventType())
	})

	t.Run("fails with invalid backup code", func(t *testing.T) {
		t.Parallel()

		secret := ReconstructTOTPSecret(
			[]byte("secret"), "goimg", "user@example.com", true, time.Now(),
		)
		_, codes, _ := GenerateBackupCodes()
		user := ReconstructUserWith2FA(
			NewUserID(), Email{}, Username{}, PasswordHash{},
			RoleUser, StatusActive, "User", "",
			time.Now(), time.Now(),
			&secret, codes, nil,
			UserTypeRegistered, nil, nil, 0,
		)

		err := user.UseBackupCode("INVALID")
		require.ErrorIs(t, err, ErrBackupCodeInvalid)
	})
}

func TestUser_UnusedBackupCodeCount(t *testing.T) {
	t.Parallel()

	secret := ReconstructTOTPSecret(
		[]byte("secret"), "goimg", "user@example.com", true, time.Now(),
	)
	plaintext, codes, _ := GenerateBackupCodes()
	user := ReconstructUserWith2FA(
		NewUserID(), Email{}, Username{}, PasswordHash{},
		RoleUser, StatusActive, "User", "",
		time.Now(), time.Now(),
		&secret, codes, nil,
		UserTypeRegistered, nil, nil, 0,
	)

	// All unused initially
	assert.Equal(t, 10, user.UnusedBackupCodeCount())

	// Use one code
	_ = user.UseBackupCode(plaintext[0])
	assert.Equal(t, 9, user.UnusedBackupCodeCount())
}

// Device Tests

func TestUser_Devices(t *testing.T) {
	t.Parallel()

	device := NewDeviceFingerprint("192.168.1.1", "user-agent")
	user := ReconstructUserWith2FA(
		NewUserID(), Email{}, Username{}, PasswordHash{},
		RoleUser, StatusActive, "User", "",
		time.Now(), time.Now(),
		nil, nil, []DeviceFingerprint{device},
		UserTypeRegistered, nil, nil, 0,
	)

	assert.Len(t, user.Devices(), 1)
}

func TestUser_TrackDevice(t *testing.T) {
	t.Parallel()

	t.Run("tracks new device", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)
		user.ClearEvents()

		ipAddress := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		fingerprint := NewDeviceFingerprint(ipAddress, userAgent)
		isTrusted := user.TrackDevice(fingerprint)

		assert.False(t, isTrusted) // New device is not trusted
		assert.Len(t, user.Devices(), 1)

		// Should emit unusual login event for new device
		events := user.Events()
		assert.Len(t, events, 1)
		assert.Equal(t, "identity.user.unusual_login", events[0].EventType())
	})

	t.Run("recognizes existing device", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		ipAddress := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		// Track once
		fingerprint := NewDeviceFingerprint(ipAddress, userAgent)
		_ = user.TrackDevice(fingerprint)
		user.ClearEvents()

		// Track again
		fingerprint2 := NewDeviceFingerprint(ipAddress, userAgent)
		isTrusted := user.TrackDevice(fingerprint2)

		assert.False(t, isTrusted) // Existing device still not trusted until explicitly trusted
		assert.Len(t, user.Devices(), 1)
		assert.Len(t, user.Events(), 0) // No event for known device
	})
}

func TestUser_TrustDevice(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("user@example.com")
	username, _ := NewUsername("testuser")
	password, _ := NewPasswordHash("ValidPassword123!")
	user, _ := NewUser(email, username, password)

	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	fingerprint := NewDeviceFingerprint(ipAddress, userAgent)
	_ = user.TrackDevice(fingerprint)
	device := user.Devices()[0]
	fingerprintHash := device.FingerprintHash()

	user.ClearEvents()

	err := user.TrustDevice(fingerprintHash)
	require.NoError(t, err)

	// Should emit device trusted event
	events := user.Events()
	assert.Len(t, events, 1)
	assert.Equal(t, "identity.user.device_trusted", events[0].EventType())
}

func TestUser_RemoveDevice(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("user@example.com")
	username, _ := NewUsername("testuser")
	password, _ := NewPasswordHash("ValidPassword123!")
	user, _ := NewUser(email, username, password)

	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	fingerprint := NewDeviceFingerprint(ipAddress, userAgent)
	_ = user.TrackDevice(fingerprint)
	device := user.Devices()[0]
	fingerprintHash := device.FingerprintHash()

	assert.Len(t, user.Devices(), 1)

	err := user.RemoveDevice(fingerprintHash)
	require.NoError(t, err)

	assert.Len(t, user.Devices(), 0)
}

func TestUser_HasTrustedDevices(t *testing.T) {
	t.Parallel()

	t.Run("returns false with no devices", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.False(t, user.HasTrustedDevices())
	})

	t.Run("returns false with only untrusted devices", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		fingerprint := NewDeviceFingerprint("192.168.1.1", "Mozilla/5.0")
		_ = user.TrackDevice(fingerprint)

		assert.False(t, user.HasTrustedDevices())
	})

	t.Run("returns true with trusted devices", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		fingerprint := NewDeviceFingerprint("192.168.1.1", "Mozilla/5.0")
		_ = user.TrackDevice(fingerprint)
		device := user.Devices()[0]
		_ = user.TrustDevice(device.FingerprintHash())

		assert.True(t, user.HasTrustedDevices())
	})
}

func TestUser_Requires2FA(t *testing.T) {
	t.Parallel()

	t.Run("requires 2FA when TOTP enabled", func(t *testing.T) {
		t.Parallel()

		secret := ReconstructTOTPSecret(
			[]byte("secret"), "goimg", "user@example.com", true, time.Now(),
		)
		user := ReconstructUserWith2FA(
			NewUserID(), Email{}, Username{}, PasswordHash{},
			RoleUser, StatusActive, "User", "",
			time.Now(), time.Now(),
			&secret, nil, nil,
			UserTypeRegistered, nil, nil, 0,
		)

		assert.True(t, user.Requires2FA())
	})

	t.Run("does not require 2FA when TOTP not enabled", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.False(t, user.Requires2FA())
	})
}

// Notification Preference Tests

func TestUser_NotificationPreferences(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("user@example.com")
	username, _ := NewUsername("testuser")
	password, _ := NewPasswordHash("ValidPassword123!")
	user, _ := NewUser(email, username, password)

	prefs := user.NotificationPreferences()
	assert.NotNil(t, prefs)
	assert.False(t, prefs.EmailEnabled()) // Default is disabled
}

func TestUser_UpdateNotificationPreferences(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("user@example.com")
	username, _ := NewUsername("testuser")
	password, _ := NewPasswordHash("ValidPassword123!")
	user, _ := NewUser(email, username, password)

	newPrefs := NewNotificationPreferences(
		true,
		map[shared.NotificationType]bool{
			shared.NotificationTypeNewFollower: true,
		},
		DigestImmediate,
	)

	user.UpdateNotificationPreferences(newPrefs)

	prefs := user.NotificationPreferences()
	assert.True(t, prefs.EmailEnabled())
	assert.Equal(t, DigestImmediate, prefs.DigestFrequency())
}

// Additional Accessor Tests

func TestUser_PasswordHash(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("user@example.com")
	username, _ := NewUsername("testuser")
	password, _ := NewPasswordHash("ValidPassword123!")
	user, _ := NewUser(email, username, password)

	assert.Equal(t, password, user.PasswordHash())
}

func TestUser_UserType(t *testing.T) {
	t.Parallel()

	t.Run("registered user", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.Equal(t, UserTypeRegistered, user.UserType())
	})

	t.Run("guest user", func(t *testing.T) {
		t.Parallel()

		user, _ := NewGuestUser("192.168.1.1")
		assert.Equal(t, UserTypeGuest, user.UserType())
	})
}

func TestUser_IPAddress(t *testing.T) {
	t.Parallel()

	t.Run("guest user has IP address", func(t *testing.T) {
		t.Parallel()

		ipAddress := "192.168.1.1"
		user, _ := NewGuestUser(ipAddress)

		assert.NotNil(t, user.IPAddress())
		assert.Equal(t, ipAddress, *user.IPAddress())
	})

	t.Run("registered user has no IP address", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.Nil(t, user.IPAddress())
	})
}

func TestUser_ExpiresAt(t *testing.T) {
	t.Parallel()

	t.Run("guest user has expiration", func(t *testing.T) {
		t.Parallel()

		user, _ := NewGuestUser("192.168.1.1")
		assert.NotNil(t, user.ExpiresAt())
	})

	t.Run("registered user has no expiration", func(t *testing.T) {
		t.Parallel()

		email, _ := NewEmail("user@example.com")
		username, _ := NewUsername("testuser")
		password, _ := NewPasswordHash("ValidPassword123!")
		user, _ := NewUser(email, username, password)

		assert.Nil(t, user.ExpiresAt())
	})
}
