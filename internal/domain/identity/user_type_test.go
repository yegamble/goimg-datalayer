package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUserType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    UserType
		wantErr bool
	}{
		{
			name:    "valid registered user type",
			input:   "registered",
			want:    UserTypeRegistered,
			wantErr: false,
		},
		{
			name:    "valid guest user type",
			input:   "guest",
			want:    UserTypeGuest,
			wantErr: false,
		},
		{
			name:    "invalid user type",
			input:   "premium",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "random string",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "uppercase registered",
			input:   "REGISTERED",
			want:    "",
			wantErr: true,
		},
		{
			name:    "mixed case guest",
			input:   "Guest",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseUserType(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user type")
				assert.Equal(t, UserType(""), got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUserType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userType UserType
		want     string
	}{
		{
			name:     "registered user type",
			userType: UserTypeRegistered,
			want:     "registered",
		},
		{
			name:     "guest user type",
			userType: UserTypeGuest,
			want:     "guest",
		},
		{
			name:     "empty user type",
			userType: UserType(""),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.userType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userType UserType
		want     bool
	}{
		{
			name:     "registered is valid",
			userType: UserTypeRegistered,
			want:     true,
		},
		{
			name:     "guest is valid",
			userType: UserTypeGuest,
			want:     true,
		},
		{
			name:     "empty is invalid",
			userType: UserType(""),
			want:     false,
		},
		{
			name:     "random string is invalid",
			userType: UserType("premium"),
			want:     false,
		},
		{
			name:     "uppercase is invalid",
			userType: UserType("REGISTERED"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.userType.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserType_IsGuest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userType UserType
		want     bool
	}{
		{
			name:     "guest type returns true",
			userType: UserTypeGuest,
			want:     true,
		},
		{
			name:     "registered type returns false",
			userType: UserTypeRegistered,
			want:     false,
		},
		{
			name:     "empty type returns false",
			userType: UserType(""),
			want:     false,
		},
		{
			name:     "invalid type returns false",
			userType: UserType("premium"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.userType.IsGuest()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserType_IsRegistered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userType UserType
		want     bool
	}{
		{
			name:     "registered type returns true",
			userType: UserTypeRegistered,
			want:     true,
		},
		{
			name:     "guest type returns false",
			userType: UserTypeGuest,
			want:     false,
		},
		{
			name:     "empty type returns false",
			userType: UserType(""),
			want:     false,
		},
		{
			name:     "invalid type returns false",
			userType: UserType("admin"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.userType.IsRegistered()
			assert.Equal(t, tt.want, got)
		})
	}
}
