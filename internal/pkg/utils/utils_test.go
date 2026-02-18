package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		given    string
		expected []string
	}{
		{
			"",
			[]string{},
		},
		{
			"a",
			[]string{"a"},
		},
		{
			"a_b",
			[]string{"a", "b"},
		},
		{
			"a_b_c",
			[]string{"a", "b", "c"},
		},
		{
			"a_b_c_d",
			[]string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.given, func(t *testing.T) {
			t.Parallel()

			got := SplitIdentifier(tt.given)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDefaultServerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		given    string
		expected string
	}{
		{
			"empty name fallback to mpc",
			"",
			"mcp",
		},
		{
			"replace dashes by underscores",
			"foo-qux-svc",
			"foo_qux_svc",
		},
		{
			"replace spaces by underscores",
			"Foo bar Svc",
			"foo_bar_svc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DefaultServerName(tt.given)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDefaultIfEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		given    string
		expected string
	}{
		{
			"empty value fallback to default",
			"",
			"default",
		},
		{
			"non-empty value returns value",
			"foo",
			"foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DefaultIfEmpty(tt.given, "default")
			assert.Equal(t, tt.expected, got)
		})
	}
}
