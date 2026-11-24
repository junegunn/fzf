package fzf

import (
	"testing"
)

func TestParseListenAddress(t *testing.T) {
	tests := []struct {
		input       string
		expected    listenAddress
		expectError bool
	}{
		// Unix socket
		{"/tmp/fzf.sock", listenAddress{"", 0, "/tmp/fzf.sock"}, false},
		{"/var/run/test.sock", listenAddress{"", 0, "/var/run/test.sock"}, false},

		// TCP with port only
		{"8080", listenAddress{"localhost", 8080, ""}, false},
		{"0", listenAddress{"localhost", 0, ""}, false},
		{"65535", listenAddress{"localhost", 65535, ""}, false},

		// TCP with host and port
		{"127.0.0.1:8080", listenAddress{"127.0.0.1", 8080, ""}, false},
		{"localhost:3000", listenAddress{"localhost", 3000, ""}, false},
		{"0.0.0.0:8080", listenAddress{"0.0.0.0", 8080, ""}, false},
		{"192.168.1.1:8080", listenAddress{"192.168.1.1", 8080, ""}, false},

		// IPv6 - note: IPv6 support may vary
		// {"[::1]:8080", listenAddress{"[::1]", 8080, ""}, false},

		// Invalid cases
		{"invalid:port", listenAddress{}, true},
		{"-1", listenAddress{}, true},
		{"65536", listenAddress{}, true},
		{"host:port:extra", listenAddress{}, true},
	}

	for _, tt := range tests {
		result, err := parseListenAddress(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("parseListenAddress(%q) expected error, got %v", tt.input, result)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseListenAddress(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if result.host != tt.expected.host ||
			result.port != tt.expected.port ||
			result.sock != tt.expected.sock {
			t.Errorf("parseListenAddress(%q) = %+v, expected %+v", tt.input, result, tt.expected)
		}
	}
}

func TestListenAddressIsLocal(t *testing.T) {
	tests := []struct {
		addr     listenAddress
		expected bool
	}{
		{listenAddress{"localhost", 8080, ""}, true},
		{listenAddress{"127.0.0.1", 8080, ""}, true},
		{listenAddress{"", 0, "/tmp/fzf.sock"}, true},
		{listenAddress{"0.0.0.0", 8080, ""}, false},
		{listenAddress{"192.168.1.1", 8080, ""}, false},
		{listenAddress{"example.com", 8080, ""}, false},
	}

	for _, tt := range tests {
		result := tt.addr.IsLocal()
		if result != tt.expected {
			t.Errorf("IsLocal(%+v) = %v, expected %v", tt.addr, result, tt.expected)
		}
	}
}
