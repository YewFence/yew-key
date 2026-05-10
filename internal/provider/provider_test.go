package provider

import (
	"reflect"
	"testing"
)

func TestParseInfisicalCustomHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "empty",
			raw:  "",
			want: nil,
		},
		{
			name: "headers",
			raw:  "Access-Client-Id=client-id Access-Client-Secret=client-secret",
			want: map[string]string{
				"Access-Client-Id":     "client-id",
				"Access-Client-Secret": "client-secret",
			},
		},
		{
			name: "value contains equals",
			raw:  "CF-Access-Jwt-Assertion=header.payload=signature",
			want: map[string]string{
				"CF-Access-Jwt-Assertion": "header.payload=signature",
			},
		},
		{
			name:    "missing separator",
			raw:     "Access-Client-Id",
			wantErr: true,
		},
		{
			name:    "missing name",
			raw:     "=client-id",
			wantErr: true,
		},
		{
			name:    "missing value",
			raw:     "Access-Client-Id=",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseInfisicalCustomHeaders(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseInfisicalCustomHeaders() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseInfisicalCustomHeaders() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseInfisicalCustomHeaders() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
