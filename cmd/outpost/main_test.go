package main

import "testing"

// TestEnvBool checks that envBool parses boolean values, uses the default for an unset variable,
// and rejects values that strconv.ParseBool does not accept.
func TestEnvBool(t *testing.T) {
	tests := []struct {
		name    string
		value   *string
		want    bool
		wantErr bool
	}{
		{name: "unset", value: nil, want: false, wantErr: false},
		{name: "true", value: ptr("true"), want: true, wantErr: false},
		{name: "one", value: ptr("1"), want: true, wantErr: false},
		{name: "false", value: ptr("false"), want: false, wantErr: false},
		{name: "yes", value: ptr("yes"), want: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != nil {
				t.Setenv("OUTPOST_TEST_BOOL", *tt.value)
			}

			got, err := envBool("OUTPOST_TEST_BOOL", false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("envBool() error = %v, want error %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("envBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ptr returns a pointer to s.
func ptr(s string) *string { return &s }
