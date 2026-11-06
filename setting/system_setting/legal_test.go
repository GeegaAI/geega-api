package system_setting

import "testing"

func TestParseLegalLanguageMap(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{name: "empty", raw: "", want: map[string]string{}},
		{name: "blank", raw: "   ", want: map[string]string{}},
		{name: "legacy plain text", raw: "# 用户协议", want: map[string]string{"": "# 用户协议"}},
		{name: "json map", raw: `{"zhCN":"正文","en":"Body"}`, want: map[string]string{"zhCN": "正文", "en": "Body"}},
		{name: "json empty object", raw: `{}`, want: map[string]string{}},
		{name: "invalid json falls back to legacy", raw: `{not json`, want: map[string]string{"": "{not json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLegalLanguageMap(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for key, want := range tt.want {
				if got[key] != want {
					t.Errorf("value[%q] = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}

func TestNormalizeLegalLanguage(t *testing.T) {
	tests := map[string]string{
		"":          "",
		"zh-CN":     "zhCN",
		"zh":        "zhCN",
		"zh-Hans":   "zhCN",
		"zhCN":      "zhCN",
		"zh-TW":     "zhTW",
		"zh-HK":     "zhTW",
		"zhTW":      "zhTW",
		"en":        "en",
		"en-US":     "en",
		"fr-FR":     "fr",
		" Japanese": "japanese",
	}
	for input, want := range tests {
		if got := NormalizeLegalLanguage(input); got != want {
			t.Errorf("NormalizeLegalLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveLegalLanguageValue(t *testing.T) {
	raw := `{"":"默认内容","en":"English content","fr":"Contenu français"}`

	tests := []struct {
		name string
		lang string
		want string
	}{
		{name: "exact language", lang: "fr", want: "Contenu français"},
		{name: "normalized language", lang: "fr-FR", want: "Contenu français"},
		{name: "falls back to default", lang: "ja", want: "默认内容"},
		{name: "empty lang falls back to default", lang: "", want: "默认内容"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveLegalLanguageValue(raw, tt.lang); got != tt.want {
				t.Errorf("ResolveLegalLanguageValue(%q) = %q, want %q", tt.lang, got, tt.want)
			}
		})
	}

	withoutDefault := `{"zhCN":"中文内容"}`
	if got := ResolveLegalLanguageValue(withoutDefault, "de"); got != "中文内容" {
		t.Errorf("expected first configured language as last resort, got %q", got)
	}

	englishOnly := `{"en":"English content"}`
	if got := ResolveLegalLanguageValue(englishOnly, "de"); got != "English content" {
		t.Errorf("expected english fallback, got %q", got)
	}

	if got := ResolveLegalLanguageValue("", "en"); got != "" {
		t.Errorf("expected empty result for empty config, got %q", got)
	}
}

func TestIsLegalValueConfigured(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "empty", raw: "", want: false},
		{name: "blank json", raw: `{"zhCN":"  "}`, want: false},
		{name: "legacy plain text", raw: "正文", want: true},
		{name: "json map", raw: `{"en":"Body"}`, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLegalValueConfigured(tt.raw); got != tt.want {
				t.Errorf("IsLegalValueConfigured(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
