package content

import "testing"

func TestExtractReading(t *testing.T) {
	tests := []struct {
		name     string
		md       string
		wantBody string
		wantSR   string
		wantRU   string
	}{
		{
			name:     "no block",
			md:       "# Урок\n\nтеория тут\n",
			wantBody: "# Урок\n\nтеория тут",
		},
		{
			name:     "sr and ru split on ---",
			md:       "# Урок\n\nтеория\n\n<!-- reading -->\nZdravo! Ja sam Milan.\n\n---\n\nПривет! Я Милан.\n<!-- /reading -->\n\nещё теория\n",
			wantBody: "# Урок\n\nтеория\n\nещё теория",
			wantSR:   "Zdravo! Ja sam Milan.",
			wantRU:   "Привет! Я Милан.",
		},
		{
			name:     "block without ru part",
			md:       "текст\n\n<!-- reading -->\nSamo srpski.\n<!-- /reading -->\n",
			wantBody: "текст",
			wantSR:   "Samo srpski.",
			wantRU:   "",
		},
		{
			name:     "no closing marker leaves md untouched",
			md:       "текст\n\n<!-- reading -->\nnedovršeno\n",
			wantBody: "текст\n\n<!-- reading -->\nnedovršeno",
			wantSR:   "",
			wantRU:   "",
		},
		{
			name:     "block at very start",
			md:       "<!-- reading -->\nPrvi pasus.\n\nDrugi pasus.\n\n---\nПервый абзац.\n\nВторой абзац.\n<!-- /reading -->\n",
			wantBody: "",
			wantSR:   "Prvi pasus.\n\nDrugi pasus.",
			wantRU:   "Первый абзац.\n\nВторой абзац.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, sr, ru := extractReading(tt.md)
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
			if sr != tt.wantSR {
				t.Errorf("sr = %q, want %q", sr, tt.wantSR)
			}
			if ru != tt.wantRU {
				t.Errorf("ru = %q, want %q", ru, tt.wantRU)
			}
		})
	}
}
