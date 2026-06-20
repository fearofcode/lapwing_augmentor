package main

import "testing"

func TestSuffixReplacementHasBareStem(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		replacedSuffix string
		want           bool
	}{
		{name: "bare key", key: "S", replacedSuffix: "S", want: true},
		{name: "bare number key", key: "#S", replacedSuffix: "S", want: true},
		{name: "bare right hand key", key: "-G", replacedSuffix: "G", want: true},
		{name: "bare numbered right hand key", key: "#-G", replacedSuffix: "-G", want: true},
		{name: "non-bare single stroke", key: "WUS", replacedSuffix: "S", want: false},
		{name: "non-bare multi stroke", key: "TKEUBGS/KWRAERZ", replacedSuffix: "Z", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suffixReplacementHasBareStem(tt.key, tt.replacedSuffix)
			if got != tt.want {
				t.Fatalf("suffixReplacementHasBareStem(%q, %q) = %v, want %v", tt.key, tt.replacedSuffix, got, tt.want)
			}
		})
	}
}

func TestLongOReplacementStroke(t *testing.T) {
	tests := []struct {
		name        string
		stroke      string
		wantStroke  string
		wantChanged bool
	}{
		{name: "short o between consonants", stroke: "SPORT", wantStroke: "SPOERT", wantChanged: true},
		{name: "proper name prefix", stroke: "#SPORT", wantStroke: "#SPOERT", wantChanged: true},
		{name: "short o without left consonant", stroke: "ORT", wantStroke: "ORT", wantChanged: false},
		{name: "short o without right consonant", stroke: "SO", wantStroke: "SO", wantChanged: false},
		{name: "already long o", stroke: "SPOERT", wantStroke: "SPOERT", wantChanged: false},
		{name: "other vowel bank containing o", stroke: "SPAOERT", wantStroke: "SPAOERT", wantChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStroke, gotChanged := longOReplacementStroke(tt.stroke)
			if gotStroke != tt.wantStroke || gotChanged != tt.wantChanged {
				t.Fatalf("longOReplacementStroke(%q) = (%q, %v), want (%q, %v)", tt.stroke, gotStroke, gotChanged, tt.wantStroke, tt.wantChanged)
			}
		})
	}
}

func TestLongOReplacementKey(t *testing.T) {
	gotKey, gotChanged := longOReplacementKey("A/SPORT/AEUGS")
	if gotKey != "A/SPOERT/AEUGS" || !gotChanged {
		t.Fatalf("longOReplacementKey(%q) = (%q, %v), want (%q, true)", "A/SPORT/AEUGS", gotKey, gotChanged, "A/SPOERT/AEUGS")
	}

	gotKey, gotChanged = longOReplacementKey("KOT/POB")
	if gotKey != "KOET/POEB" || !gotChanged {
		t.Fatalf("longOReplacementKey(%q) = (%q, %v), want (%q, true)", "KOT/POB", gotKey, gotChanged, "KOET/POEB")
	}
}

func TestFinalEUToAOEReplacementStroke(t *testing.T) {
	tests := []struct {
		name        string
		stroke      string
		wantStroke  string
		wantChanged bool
	}{
		{name: "left consonants eu", stroke: "KEU", wantStroke: "KAOE", wantChanged: true},
		{name: "cluster eu", stroke: "TPHEU", wantStroke: "TPHAOE", wantChanged: true},
		{name: "bare eu", stroke: "EU", wantStroke: "EU", wantChanged: false},
		{name: "already aoe", stroke: "KAOE", wantStroke: "KAOE", wantChanged: false},
		{name: "eu with right consonants", stroke: "KEUPB", wantStroke: "KEUPB", wantChanged: false},
		{name: "right hand only", stroke: "-PBEU", wantStroke: "-PBEU", wantChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStroke, gotChanged := finalEUToAOEReplacementStroke(tt.stroke)
			if gotStroke != tt.wantStroke || gotChanged != tt.wantChanged {
				t.Fatalf("finalEUToAOEReplacementStroke(%q) = (%q, %v), want (%q, %v)", tt.stroke, gotStroke, gotChanged, tt.wantStroke, tt.wantChanged)
			}
		})
	}
}

func TestFinalEUToAOEReplacementKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantKey     string
		wantChanged bool
	}{
		{name: "turkey shape", key: "TEUR/KEU", wantKey: "TEUR/KAOE", wantChanged: true},
		{name: "does not rewrite earlier stroke", key: "SEU/KEU", wantKey: "SEU/KAOE", wantChanged: true},
		{name: "single stroke", key: "KEU", wantKey: "KEU", wantChanged: false},
		{name: "bare final eu", key: "TEUR/EU", wantKey: "TEUR/EU", wantChanged: false},
		{name: "non-final eu unchanged", key: "KEU/TUR", wantKey: "KEU/TUR", wantChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotChanged := finalEUToAOEReplacementKey(tt.key)
			if gotKey != tt.wantKey || gotChanged != tt.wantChanged {
				t.Fatalf("finalEUToAOEReplacementKey(%q) = (%q, %v), want (%q, %v)", tt.key, gotKey, gotChanged, tt.wantKey, tt.wantChanged)
			}
		})
	}
}

func TestInitialKHToKPHReplacementStroke(t *testing.T) {
	tests := []struct {
		name        string
		stroke      string
		wantStroke  string
		wantChanged bool
	}{
		{name: "initial kh", stroke: "KHAO", wantStroke: "KPHAO", wantChanged: true},
		{name: "proper name prefix", stroke: "#KHAO", wantStroke: "#KPHAO", wantChanged: true},
		{name: "initial khr", stroke: "KHRAO", wantStroke: "KPHRAO", wantChanged: true},
		{name: "already kph", stroke: "KPHAO", wantStroke: "KPHAO", wantChanged: false},
		{name: "kh not at initial left edge", stroke: "SKHAO", wantStroke: "SKHAO", wantChanged: false},
		{name: "right hand only", stroke: "-FP", wantStroke: "-FP", wantChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStroke, gotChanged := initialKHToKPHReplacementStroke(tt.stroke)
			if gotStroke != tt.wantStroke || gotChanged != tt.wantChanged {
				t.Fatalf("initialKHToKPHReplacementStroke(%q) = (%q, %v), want (%q, %v)", tt.stroke, gotStroke, gotChanged, tt.wantStroke, tt.wantChanged)
			}
		})
	}
}

func TestInitialKHToKPHReplacementKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantKey     string
		wantChanged bool
	}{
		{name: "single stroke chao shape", key: "KHAO", wantKey: "KPHAO", wantChanged: true},
		{name: "multi stroke", key: "A/KHAO", wantKey: "A/KPHAO", wantChanged: true},
		{name: "rewrites every matching stroke", key: "KHAO/KHEU", wantKey: "KPHAO/KPHEU", wantChanged: true},
		{name: "no matching stroke", key: "SKHAO", wantKey: "SKHAO", wantChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotChanged := initialKHToKPHReplacementKey(tt.key)
			if gotKey != tt.wantKey || gotChanged != tt.wantChanged {
				t.Fatalf("initialKHToKPHReplacementKey(%q) = (%q, %v), want (%q, %v)", tt.key, gotKey, gotChanged, tt.wantKey, tt.wantChanged)
			}
		})
	}
}
