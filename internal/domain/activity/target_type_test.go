package activity

import (
	"testing"
)

func TestTargetType_String(t *testing.T) {
	if got := TargetTypeImage.String(); got != "image" {
		t.Errorf("TargetTypeImage.String() = %v, want image", got)
	}
}

func TestTargetType_ParseTargetType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TargetType
		wantErr bool
	}{
		{"Valid image", "image", TargetTypeImage, false},
		{"Valid user", "user", TargetTypeUser, false},
		{"Valid album", "album", TargetTypeAlbum, false},
		{"Valid comment", "comment", TargetTypeComment, false},
		{"Invalid type", "invalid", "", true},
		{"Empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTargetType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTargetType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTargetType() = %v, want %v", got, tt.want)
			}
		})
	}
}
