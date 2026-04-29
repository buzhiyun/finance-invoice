package task

import (
	"testing"
)

func TestFieldMatchesPDFText(t *testing.T) {
	pdfText := []string{
		"发票号码：26622000000179523856",
		"开票日期：2026年03月17日",
		"名称：安徽七天网络科技有限公司",
		"统一社会信用代码/纳税人识别号：9134010007564947X2",
		"名称：甘肃中启网络科技有限公司",
		"统一社会信用代码/纳税人识别号：91620104MAELGLFD88",
		"合计 ¥7,916.73 ¥79.17",
		"¥7,995.90",
		"开票人：刘见华",
	}

	tests := []struct {
		value string
		want  bool
	}{
		{"26622000000179523856", true},
		{"2662200000179523856", false},   // missing a zero
		{"2026年03月17日", true},
		{"安徽七天网络科技有限公司", true},
		{"9134010007564947X2", true},
		{"甘肃中启网络科技有限公司", true},
		{"91620104MAELGLFD88", true},
		{"7916.73", true},
		{"¥7995.90", true},               // with ¥ prefix
		{"7995.90", true},                // without ¥ prefix
		{"刘见华", true},
		{"1%", false},                     // not in PDF text
		{"", true},                        // empty value - don't validate
	}

	for _, tt := range tests {
		got := FieldMatchesPDFText(tt.value, pdfText)
		if got != tt.want {
			t.Errorf("FieldMatchesPDFText(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestFieldMatchesPDFText_NoPDFText(t *testing.T) {
	// No PDF text available - should return true (can't validate)
	if !FieldMatchesPDFText("some value", nil) {
		t.Error("FieldMatchesPDFText with nil pdfText should return true")
	}
	if !FieldMatchesPDFText("some value", []string{}) {
		t.Error("FieldMatchesPDFText with empty pdfText should return true")
	}
}

func TestFieldMatchesPDFText_SpaceHandling(t *testing.T) {
	// PDF text might have spaces in numbers
	pdfText := []string{
		"2 662 2000 0001 7952 3856",
	}

	if !FieldMatchesPDFText("26622000000179523856", pdfText) {
		t.Error("Should match when PDF text has spaces in number")
	}
}
