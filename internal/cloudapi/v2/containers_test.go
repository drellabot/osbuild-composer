package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseContainerRef(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantReg  string
		wantRepo string
		wantTag  string
		wantErr  bool
	}{
		{
			name:     "quay.io with tag",
			source:   "quay.io/fedora/fedora:latest",
			wantReg:  "quay.io",
			wantRepo: "fedora/fedora",
			wantTag:  "latest",
		},
		{
			name:     "docker.io library image",
			source:   "docker.io/library/nginx:alpine",
			wantReg:  "registry-1.docker.io",
			wantRepo: "library/nginx",
			wantTag:  "alpine",
		},
		{
			name:     "docker.io short form",
			source:   "docker.io/nginx",
			wantReg:  "registry-1.docker.io",
			wantRepo: "library/nginx",
			wantTag:  "latest",
		},
		{
			name:     "ghcr.io no tag",
			source:   "ghcr.io/owner/repo",
			wantReg:  "ghcr.io",
			wantRepo: "owner/repo",
			wantTag:  "latest",
		},
		{
			name:     "digest reference",
			source:   "quay.io/org/img@sha256:abc123",
			wantReg:  "quay.io",
			wantRepo: "org/img",
			wantTag:  "sha256:abc123",
		},
		{
			name:    "missing registry",
			source:  "nginx",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := parseContainerRef(tt.source)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantReg, ref.registry)
			require.Equal(t, tt.wantRepo, ref.repository)
			require.Equal(t, tt.wantTag, ref.tag)
		})
	}
}
