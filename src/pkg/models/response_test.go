package models

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewUserLoginResponse(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		username string
		role     string
		until    string
		expected UserLoginResponse
	}{
		{
			name:     "valid_user_login_response",
			username: "alice@example.com",
			role:     "cluster-admin",
			until:    "2023-12-31T23:59:59Z",
			expected: UserLoginResponse{
				Username: "alice@example.com",
				Role:     "cluster-admin",
				Until:    "2023-12-31T23:59:59Z",
			},
		},
		{
			name:     "empty_username",
			username: "",
			role:     "viewer",
			until:    "2023-12-31T23:59:59Z",
			expected: UserLoginResponse{
				Username: "",
				Role:     "viewer",
				Until:    "2023-12-31T23:59:59Z",
			},
		},
		{
			name:     "empty_role",
			username: "bob@example.com",
			role:     "",
			until:    "2023-12-31T23:59:59Z",
			expected: UserLoginResponse{
				Username: "bob@example.com",
				Role:     "",
				Until:    "2023-12-31T23:59:59Z",
			},
		},
		{
			name:     "empty_until",
			username: "charlie@example.com",
			role:     "editor",
			until:    "",
			expected: UserLoginResponse{
				Username: "charlie@example.com",
				Role:     "editor",
				Until:    "",
			},
		},
		{
			name:     "all_empty_fields",
			username: "",
			role:     "",
			until:    "",
			expected: UserLoginResponse{
				Username: "",
				Role:     "",
				Until:    "",
			},
		},
		{
			name:     "special_characters_in_username",
			username: "user+tag@example-domain.co.uk",
			role:     "namespace-admin",
			until:    "2024-01-15T12:30:45Z",
			expected: UserLoginResponse{
				Username: "user+tag@example-domain.co.uk",
				Role:     "namespace-admin",
				Until:    "2024-01-15T12:30:45Z",
			},
		},
		{
			name:     "hyphenated_role",
			username: "dev@company.com",
			role:     "read-only-user",
			until:    "2024-06-30T18:00:00Z",
			expected: UserLoginResponse{
				Username: "dev@company.com",
				Role:     "read-only-user",
				Until:    "2024-06-30T18:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := NewUserLoginResponse(tt.username, tt.role, tt.until)

			if got.Username != tt.expected.Username {
				t.Errorf("Username = %q, want %q", got.Username, tt.expected.Username)
			}

			if got.Role != tt.expected.Role {
				t.Errorf("Role = %q, want %q", got.Role, tt.expected.Role)
			}

			if got.Until != tt.expected.Until {
				t.Errorf("Until = %q, want %q", got.Until, tt.expected.Until)
			}
		})
	}
}

func TestUserLoginResponse_JSON_Marshaling(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		input    UserLoginResponse
		expected string
	}{
		{
			name: "complete_response",
			input: UserLoginResponse{
				Username: "alice@example.com",
				Role:     "cluster-admin",
				Until:    "2023-12-31T23:59:59Z",
			},
			expected: `{"username":"alice@example.com","role":"cluster-admin","until":"2023-12-31T23:59:59Z"}`,
		},
		{
			name: "empty_fields",
			input: UserLoginResponse{
				Username: "",
				Role:     "",
				Until:    "",
			},
			expected: `{"username":"","role":"","until":""}`,
		},
		{
			name: "partial_fields",
			input: UserLoginResponse{
				Username: "user@domain.com",
				Role:     "",
				Until:    "2024-01-01T00:00:00Z",
			},
			expected: `{"username":"user@domain.com","role":"","until":"2024-01-01T00:00:00Z"}`,
		},
		{
			name: "special_characters",
			input: UserLoginResponse{
				Username: "user+test@sub.domain.co.uk",
				Role:     "namespace-admin",
				Until:    "2024-12-31T23:59:59.999Z",
			},
			expected: `{"username":"user+test@sub.domain.co.uk","role":"namespace-admin","until":"2024-12-31T23:59:59.999Z"}`,
		},
		{
			name: "unicode_characters",
			input: UserLoginResponse{
				Username: "用户@example.com",
				Role:     "管理员",
				Until:    "2024-06-15T12:00:00Z",
			},
			expected: `{"username":"用户@example.com","role":"管理员","until":"2024-06-15T12:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			// Test marshaling
			gotJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if string(gotJSON) != tt.expected {
				t.Errorf("json.Marshal() = %s, want %s", string(gotJSON), tt.expected)
			}
		})
	}
}

func TestUserLoginResponse_JSON_Unmarshaling(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		input    string
		expected UserLoginResponse
	}{
		{
			name:  "complete_json",
			input: `{"username":"alice@example.com","role":"cluster-admin","until":"2023-12-31T23:59:59Z"}`,
			expected: UserLoginResponse{
				Username: "alice@example.com",
				Role:     "cluster-admin",
				Until:    "2023-12-31T23:59:59Z",
			},
		},
		{
			name:  "empty_fields",
			input: `{"username":"","role":"","until":""}`,
			expected: UserLoginResponse{
				Username: "",
				Role:     "",
				Until:    "",
			},
		},
		{
			name:  "missing_fields",
			input: `{}`,
			expected: UserLoginResponse{
				Username: "",
				Role:     "",
				Until:    "",
			},
		},
		{
			name:  "partial_fields",
			input: `{"username":"user@example.com"}`,
			expected: UserLoginResponse{
				Username: "user@example.com",
				Role:     "",
				Until:    "",
			},
		},
		{
			name:  "extra_fields_ignored",
			input: `{"username":"test@example.com","role":"admin","until":"2024-01-01T00:00:00Z","extra":"ignored"}`,
			expected: UserLoginResponse{
				Username: "test@example.com",
				Role:     "admin",
				Until:    "2024-01-01T00:00:00Z",
			},
		},
		{
			name:  "unicode_characters",
			input: `{"username":"用户@example.com","role":"管理员","until":"2024-06-15T12:00:00Z"}`,
			expected: UserLoginResponse{
				Username: "用户@example.com",
				Role:     "管理员",
				Until:    "2024-06-15T12:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			var got UserLoginResponse
			err := json.Unmarshal([]byte(tt.input), &got)
			if err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.Username != tt.expected.Username {
				t.Errorf("Username = %q, want %q", got.Username, tt.expected.Username)
			}

			if got.Role != tt.expected.Role {
				t.Errorf("Role = %q, want %q", got.Role, tt.expected.Role)
			}

			if got.Until != tt.expected.Until {
				t.Errorf("Until = %q, want %q", got.Until, tt.expected.Until)
			}
		})
	}
}

func TestUserLoginResponse_JSON_RoundTrip(t *testing.T) {
	t.Helper()

	tests := []UserLoginResponse{
		{
			Username: "alice@example.com",
			Role:     "cluster-admin",
			Until:    "2023-12-31T23:59:59Z",
		},
		{
			Username: "bob@company.org",
			Role:     "namespace-viewer",
			Until:    "2024-06-15T14:30:00Z",
		},
		{
			Username: "",
			Role:     "",
			Until:    "",
		},
		{
			Username: "special+user@sub.domain.co.uk",
			Role:     "admin-role",
			Until:    "2025-01-01T00:00:00.000Z",
		},
	}

	for i, original := range tests {
		t.Run(fmt.Sprintf("round_trip_%d", i), func(t *testing.T) {
			t.Helper()

			// Marshal to JSON
			jsonData, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Unmarshal back to struct
			var got UserLoginResponse
			err = json.Unmarshal(jsonData, &got)
			if err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			// Compare results
			if got.Username != original.Username {
				t.Errorf("Username = %q, want %q", got.Username, original.Username)
			}

			if got.Role != original.Role {
				t.Errorf("Role = %q, want %q", got.Role, original.Role)
			}

			if got.Until != original.Until {
				t.Errorf("Until = %q, want %q", got.Until, original.Until)
			}
		})
	}
}

func TestUserLoginResponse_JSON_InvalidFormat(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid_json_syntax",
			input: `{"username": invalid}`,
		},
		{
			name:  "unclosed_braces",
			input: `{"username":"test"`,
		},
		{
			name:  "invalid_quotes",
			input: `{'username':'test'}`,
		},
		{
			name:  "array_instead_of_object",
			input: `["username", "role", "until"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			var got UserLoginResponse
			err := json.Unmarshal([]byte(tt.input), &got)

			if err == nil {
				t.Errorf("expected error for invalid JSON: %s", tt.input)
			}
		})
	}
}
