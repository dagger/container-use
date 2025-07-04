package main

import (
	"context"
	"testing"

	"github.com/dagger/container-use/environment"
)

func TestEnvironmentFromArgs(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name          string
		arg           string
		repo          repoLister
		expected      string
		expectedError bool
	}{
		{
			name:          "arg defined and not environments",
			arg:           "fancy-mallard",
			repo:          &repo{},
			expected:      "fancy-mallard",
			expectedError: false,
		},
		{
			name: "arg defined and environments",
			arg:  "fancy-mallard",
			repo: &repo{
				list: []*environment.EnvironmentInfo{
					{ID: "foo"},
					{ID: "bar"},
				},
			},
			expected:      "fancy-mallard",
			expectedError: false,
		},
		{
			name:          "no arg, no environment",
			arg:           "",
			repo:          &repo{},
			expected:      "",
			expectedError: true,
		},
		{
			name: "no arg, one environment",
			arg:  "",
			repo: &repo{
				list: []*environment.EnvironmentInfo{
					{ID: "fancy-mallard"},
				},
			},
			expected:      "fancy-mallard",
			expectedError: false,
		},
		{
			name: "no arg, more than one environment",
			arg:  "",
			repo: &repo{
				list: []*environment.EnvironmentInfo{
					{ID: "fancy-mallard"},
					{ID: "bar"},
				},
			},
			expected:      "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := envOrDefault(ctx, tt.arg, tt.repo)
			if (err != nil) != tt.expectedError {
				t.Errorf("envOrDefault() error = %v, wantErr %v", err, tt.expectedError)
				return
			}
			if env != tt.expected {
				t.Errorf("envOrDefault() = %v, want %v", env, tt.expected)
			}
		})
	}
}

type repo struct {
	list []*environment.EnvironmentInfo
}

func (r *repo) List(_ context.Context) ([]*environment.EnvironmentInfo, error) {
	return r.list, nil
}
