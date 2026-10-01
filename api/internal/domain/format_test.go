package domain

import "testing"

func TestFormatReach(t *testing.T) {
	cases := []struct {
		lakh float64
		want string
	}{
		{34.2, "34.2L"},
		{100, "1.00Cr"},
		{123, "1.23Cr"},
		{0, "0.0L"},
	}
	for _, c := range cases {
		if got := FormatReach(c.lakh); got != c.want {
			t.Fatalf("FormatReach(%v) = %q, want %q", c.lakh, got, c.want)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		rupees int64
		want   string
	}{
		{900, "₹900"},
		{44_000, "₹44K"},
		{4_40_000, "₹4.4L"},
		{1_23_00_000, "₹1.23Cr"},
	}
	for _, c := range cases {
		if got := FormatMoney(c.rupees); got != c.want {
			t.Fatalf("FormatMoney(%v) = %q, want %q", c.rupees, got, c.want)
		}
	}
}

func TestFormatIndex(t *testing.T) {
	if got := FormatIndex(1.4); got != "1.4x" {
		t.Fatalf("FormatIndex(1.4) = %q, want 1.4x", got)
	}
}

func TestFormatCPM(t *testing.T) {
	if got := FormatCPM(0); got != "—" {
		t.Fatalf("FormatCPM(0) = %q, want —", got)
	}
	if got := FormatCPM(123.6); got != "₹124" {
		t.Fatalf("FormatCPM(123.6) = %q, want ₹124", got)
	}
}
