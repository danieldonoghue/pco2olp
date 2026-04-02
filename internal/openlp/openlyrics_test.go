package openlp

import (
	"strings"
	"testing"
)

func TestParsePCOLyrics(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantVerseCount int
		wantTags       []string
	}{
		{
			name: "standard song",
			input: `VERSE 1
Amazing grace how sweet the sound
That saved a wretch like me

CHORUS
Through many dangers toils and snares
I have already come

VERSE 2
Twas grace that taught my heart to fear
And grace my fears relieved`,
			wantVerseCount: 3,
			wantTags:       []string{"v1", "c1", "v2"},
		},
		{
			name: "with bridge and pre-chorus",
			input: `VERSE 1
Line one
Line two

PRE-CHORUS
Build up line

CHORUS
Chorus line

BRIDGE
Bridge line`,
			wantVerseCount: 4,
			wantTags:       []string{"v1", "p1", "c1", "b1"},
		},
		{
			name:           "empty lyrics",
			input:          "",
			wantVerseCount: 0,
			wantTags:       nil,
		},
		{
			name: "text without labels",
			input: `Some text without any labels
Another line here`,
			wantVerseCount: 1,
			wantTags:       []string{"v1"},
		},
		{
			name: "case insensitive labels",
			input: `verse 1
First verse

Chorus
Chorus text`,
			wantVerseCount: 2,
			wantTags:       []string{"v1", "c1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verses, order := ParsePCOLyrics(tt.input)
			if len(verses) != tt.wantVerseCount {
				t.Errorf("got %d verses, want %d", len(verses), tt.wantVerseCount)
			}
			if tt.wantTags != nil {
				if len(order) != len(tt.wantTags) {
					t.Fatalf("got %d order entries, want %d: %v", len(order), len(tt.wantTags), order)
				}
				for i, tag := range tt.wantTags {
					if order[i] != tag {
						t.Errorf("order[%d] = %q, want %q", i, order[i], tag)
					}
				}
			}
		})
	}
}

func TestGenerateOpenLyrics(t *testing.T) {
	verses := []VerseContent{
		{Tag: "v1", Lines: []string{"Amazing grace how sweet the sound", "That saved a wretch like me"}},
		{Tag: "c1", Lines: []string{"Through many dangers toils and snares"}},
	}

	xml, err := GenerateOpenLyrics("Amazing Grace", "John Newton", "(c) Public Domain", "12345", verses, []string{"v1", "c1"})
	if err != nil {
		t.Fatalf("GenerateOpenLyrics: %v", err)
	}

	checks := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`xmlns="http://openlyrics.info/namespace/2009/song"`,
		`<title>Amazing Grace</title>`,
		`<author>John Newton</author>`,
		`<ccliNo>12345</ccliNo>`,
		`<verseOrder>v1 c1</verseOrder>`,
		`name="v1"`,
		`name="c1"`,
		`Amazing grace how sweet the sound<br/>That saved a wretch like me`,
	}

	for _, check := range checks {
		if !strings.Contains(xml, check) {
			t.Errorf("XML missing %q\nGot:\n%s", check, xml)
		}
	}
}

func TestVerseTagToUpper(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"v1", "V1"},
		{"c1", "C1"},
		{"b1", "B1"},
		{"p1", "P1"},
	}
	for _, tt := range tests {
		if got := VerseTagToUpper(tt.input); got != tt.want {
			t.Errorf("VerseTagToUpper(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
