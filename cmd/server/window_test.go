package main

import (
	"reflect"
	"testing"
)

func TestClientArgsWithoutWindow(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"nil", nil, []string{}},
		{"no window flag", []string{"config", "client"}, []string{"config", "client"}},
		{"single window flag", []string{"config", "client", "--window"}, []string{"config", "client"}},
		{"window first", []string{"--window", "config", "client"}, []string{"config", "client"}},
		{"multiple window flags", []string{"--window", "config", "--window", "client", "--window"}, []string{"config", "client"}},
		{"similar flag kept", []string{"--windowed", "config"}, []string{"--windowed", "config"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientArgsWithoutWindow(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientArgsWithoutWindow(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
