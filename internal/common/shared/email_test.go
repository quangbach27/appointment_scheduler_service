package shared_test

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"scheduler/internal/common/shared"
)

func TestNewEmail(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "returns email when input is valid",
			input:   "user@example.com",
			want:    "user@example.com",
			wantErr: false,
		},
		{
			name:    "returns email when input has plus tag and subdomain",
			input:   "a.b+c@sub.example.co",
			want:    "a.b+c@sub.example.co",
			wantErr: false,
		},
		{
			name:    "returns error when input is empty",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "returns error when input is missing @",
			input:   "plainaddress",
			want:    "",
			wantErr: true,
		},
		{
			name:    "returns error when input is missing local part",
			input:   "@example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "returns error when input is missing domain",
			input:   "user@",
			want:    "",
			wantErr: true,
		},
		{
			name:    "returns error when input contains space",
			input:   "user example.com",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			email, err := shared.NewEmail(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, email.IsZero())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.input, email.String())
			assert.False(t, email.IsZero())
		})
	}
}

func TestScan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:    "returns email when src is string",
			input:   "user@example.com",
			want:    "user@example.com",
			wantErr: false,
		},
		{
			name:    "returns email when src is bytes",
			input:   []byte("user@example.com"),
			want:    "user@example.com",
			wantErr: false,
		},
		{
			name:    "returns empty when src is nil",
			input:   nil,
			want:    "",
			wantErr: false,
		},
		{
			name:    "returns errors when src is unsupported type",
			input:   123,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var e shared.Email

			err := e.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, e.String())
		})
	}
}

func TestValue(t *testing.T) {
	t.Parallel()

	email, err := shared.NewEmail("user@gmail.com")
	require.NoError(t, err)

	v, err := email.Value()
	require.NoError(t, err)
	assert.Equal(t, email.String(), v)
}
