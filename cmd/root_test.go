package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSkipTokenRefresh(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path []string // command chain from root to leaf
		want bool
	}{
		{name: "auth command", path: []string{"auth"}, want: true},
		{name: "config command", path: []string{"config"}, want: true},
		{name: "version command", path: []string{"version"}, want: true},
		{name: "help command", path: []string{"help"}, want: true},
		{name: "completion subcommand", path: []string{"completion", "zsh"}, want: true},
		{name: "get command", path: []string{"get"}, want: false},
		{name: "list command", path: []string{"list"}, want: false},
		{name: "create command", path: []string{"create"}, want: false},
		{name: "root only", path: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := &cobra.Command{Use: "gotion"}
			leaf := root
			for _, name := range tt.path {
				child := &cobra.Command{Use: name}
				leaf.AddCommand(child)
				leaf = child
			}

			if got := skipTokenRefresh(leaf); got != tt.want {
				t.Errorf("skipTokenRefresh(%v) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
