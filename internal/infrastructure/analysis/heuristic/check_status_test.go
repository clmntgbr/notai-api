package heuristic

import "testing"

func TestOkAndSkippedSignals_DoNotAffectScoringShape(t *testing.T) {
	ok := okSignal("frequency", "Frequency spectrum within normal range")
	if ok.Type != "meta" || ok.Code != "frequency_ok" || ok.Weight != 0 {
		t.Fatalf("unexpected ok signal: %+v", ok)
	}

	skipped := skippedSignal("jpeg_quant", "not applicable")
	if skipped.Type != "meta" || skipped.Code != "jpeg_quant_skipped" || skipped.Weight != 0 {
		t.Fatalf("unexpected skipped signal: %+v", skipped)
	}
}

func TestJPEGQuantCheck_SkipsNonJPEG(t *testing.T) {
	sigs, err := (JPEGQuantCheck{}).Run(t.Context(), CheckInput{Format: "webp"}, CheckParams{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) != 1 || sigs[0].Code != "jpeg_quant_skipped" {
		t.Fatalf("expected jpeg_quant_skipped, got %+v", sigs)
	}
}

func TestExifCheck_SkippedForWebP(t *testing.T) {
	sigs, err := (ExifCheck{}).Run(t.Context(), CheckInput{Format: "webp", RawBytes: []byte("not-c2pa")}, CheckParams{
		Enabled: true,
		Weights: map[string]float64{"exif_missing": 0.15, "c2pa_generative": 0.95},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) != 1 || sigs[0].Code != "exif_skipped" {
		t.Fatalf("expected exif_skipped, got %+v", sigs)
	}
}
