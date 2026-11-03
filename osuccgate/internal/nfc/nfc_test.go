package nfc

import (
	"testing"
)

func TestSelectPreferredReader(t *testing.T) {
	tests := []struct {
		name     string
		readers  []string
		expected string
	}{
		{
			name:     "empty readers",
			readers:  []string{},
			expected: "",
		},
		{
			name:     "single generic reader",
			readers:  []string{"Generic Smart Card Reader 00 00"},
			expected: "Generic Smart Card Reader 00 00",
		},
		{
			name: "prefer RC-S380 over generic reader",
			readers: []string{
				"Generic USB Smart Card Reader 00 00",
				"Sony FeliCa Port/PaSoRi 3.0 (RC-S380) 00 00",
			},
			expected: "Sony FeliCa Port/PaSoRi 3.0 (RC-S380) 00 00",
		},
		{
			name: "prefer PaSoRi keyword",
			readers: []string{
				"Other Reader 01",
				"SONY PaSoRi 3.0",
			},
			expected: "SONY PaSoRi 3.0",
		},
		{
			name: "prefer Sony keyword",
			readers: []string{
				"Alcor Micro USB Smart Card Reader",
				"Sony Contactless Reader",
			},
			expected: "Sony Contactless Reader",
		},
		{
			name: "prefer FeliCa keyword",
			readers: []string{
				"Virtual Smart Card",
				"FeliCa Reader 00",
			},
			expected: "FeliCa Reader 00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SelectPreferredReader(tc.readers)
			if got != tc.expected {
				t.Errorf("SelectPreferredReader(%v) = %q, want %q", tc.readers, got, tc.expected)
			}
		})
	}
}

func TestNormalizeCardID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0123456789abcdef", "0123456789ABCDEF"},
		{";1234567890?\r\n", "1234567890"},
		{"%abcdef?\n", "ABCDEF"},
		{"  0114aabbccddeeff  ", "0114AABBCCDDEEFF"},
	}

	for _, tc := range tests {
		got := NormalizeCardID(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeCardID(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

