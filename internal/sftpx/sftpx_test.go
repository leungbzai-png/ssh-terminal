package sftpx

import "testing"

func TestIsProbablyText(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", []byte{}, true},
		{"ascii", []byte("hello world\n"), true},
		{"utf8", []byte("héllo, 世界\n"), true},
		{"json", []byte(`{"a":1,"b":"x"}`), true},
		{"nul byte", []byte("abc\x00def"), false},
		{"invalid utf8", []byte{0xff, 0xfe, 0xfd}, false},
		{"binary-ish", append([]byte("PK\x03\x04"), 0x00, 0x01), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsProbablyText(c.data); got != c.want {
				t.Errorf("IsProbablyText(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}
