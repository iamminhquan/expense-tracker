package format_test

import (
	"testing"

	"expensetracker/internal/format"
)

func TestDeviceLabel(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want string
	}{
		{
			name: "empty",
			ua:   "",
			want: "Unknown device",
		},
		{
			name: "chrome on windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			want: "Chrome on Windows",
		},
		{
			name: "safari on iphone",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			want: "Safari on iPhone",
		},
		{
			name: "firefox on linux",
			ua:   "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0",
			want: "Firefox on Linux",
		},
		{
			name: "chrome on android",
			ua:   "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
			want: "Chrome on Android",
		},
		{
			name: "safari on mac",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
			want: "Safari on macOS",
		},
		{
			name: "edge on windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0",
			want: "Edge on Windows",
		},
		{
			name: "unrecognized string falls back to the raw UA",
			ua:   "SomeCustomBot/1.0 (+https://example.com/bot)",
			want: "SomeCustomBot/1.0 (+https://example.com/bot)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := format.DeviceLabel(tc.ua); got != tc.want {
				t.Errorf("DeviceLabel(%q) = %q, want %q", tc.ua, got, tc.want)
			}
		})
	}
}
