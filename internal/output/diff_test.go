package output

import (
	"bytes"
	"strings"
	"testing"
)

const sampleDiff = `diff --git a/internal/kafka/consumer.go b/internal/kafka/consumer.go
index abc1234..def5678 100644
--- a/internal/kafka/consumer.go
+++ b/internal/kafka/consumer.go
@@ -10,7 +10,8 @@ import (
 )

 func NewConsumer(brokers []string) *Consumer {
-	return &Consumer{brokers: brokers}
+	c := &Consumer{brokers: brokers, timeout: 30}
+	return c
 }

 func (c *Consumer) Read() (Msg, error) {
diff --git a/README.md b/README.md
index 111..222 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
 # Service
+Upgraded to kafka 3.8.
 Run with ./bin/svc.
-Old instructions here.
`

func TestRenderDiff_NoColor_IsIdempotent(t *testing.T) {
	var out bytes.Buffer
	if err := RenderDiff(strings.NewReader(sampleDiff), &out, ColorNever); err != nil {
		t.Fatalf("render: %v", err)
	}
	if out.String() != sampleDiff {
		t.Fatalf("plain render should round-trip; got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("plain render must not emit ANSI escapes")
	}
}

func TestRenderDiff_AlwaysEmitsAnsi(t *testing.T) {
	var out bytes.Buffer
	if err := RenderDiff(strings.NewReader(sampleDiff), &out, ColorAlways); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := out.String()
	for _, want := range []string{ansiGreen, ansiRed, ansiCyan, ansiBlue} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected ANSI %q in coloured output", want)
		}
	}
}

func TestRenderDiffStat_Counts(t *testing.T) {
	var out bytes.Buffer
	if err := RenderDiffStat(strings.NewReader(sampleDiff), &out, ColorNever); err != nil {
		t.Fatalf("stat: %v", err)
	}
	got := out.String()
	wantFragments := []string{
		"internal/kafka/consumer.go",
		"README.md",
		"2 file(s) changed",
		"3 insertion(s)(+)",
		"2 deletion(s)(-)",
	}
	for _, w := range wantFragments {
		if !strings.Contains(got, w) {
			t.Fatalf("stat output missing %q; got:\n%s", w, got)
		}
	}
}

func TestParseColorMode(t *testing.T) {
	cases := map[string]ColorMode{
		"":       ColorAuto,
		"auto":   ColorAuto,
		"always": ColorAlways,
		"ALWAYS": ColorAlways,
		"never":  ColorNever,
		"no":     ColorNever,
	}
	for in, want := range cases {
		if got := ParseColorMode(in); got != want {
			t.Errorf("ParseColorMode(%q)=%v want %v", in, got, want)
		}
	}
}
