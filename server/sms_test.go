package main

import (
	"regexp"
	"strings"
	"testing"
)

var placeholderRe = regexp.MustCompile(`\{([A-Z]+)\}`)

func TestSmsTemplateDefaultsAreConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range smsTemplateDefs {
		if seen[d.Type] {
			t.Fatalf("duplicate template type %q", d.Type)
		}
		seen[d.Type] = true
		for _, lang := range []string{"sw", "en"} {
			body := d.Body[lang]
			if body == "" {
				t.Fatalf("%s has no %s body", d.Type, lang)
			}
			if len([]rune(body)) > 320 {
				t.Errorf("%s/%s is %d chars — keep default SMS short", d.Type, lang, len([]rune(body)))
			}
			for _, m := range placeholderRe.FindAllStringSubmatch(body, -1) {
				ok := m[1] == "BRAND" // global: filled in for every template (see fillSms)
				for _, v := range d.Vars {
					ok = ok || v == m[1]
				}
				if !ok {
					t.Errorf("%s/%s uses {%s}, which is not declared in Vars", d.Type, lang, m[1])
				}
			}
		}
	}
	// every transaction type the handlers pass to txSmsText must resolve
	for _, typ := range []string{"contribution", "share", "social_fund", "loan_repayment", "loan_disbursement", "fine"} {
		if findSmsTemplate(typ) == nil {
			t.Errorf("no template for transaction type %q", typ)
		}
	}
}

func TestFillSms(t *testing.T) {
	d := findSmsTemplate("contribution")
	got := fillSms(d.Body["sw"], map[string]string{"JINA": "Asha", "KIKUNDI": "Umoja", "KIASI": "TZS 5,000", "TAREHE": "12/09/2026"})
	if strings.Contains(got, "{") {
		t.Fatalf("unreplaced placeholder in %q", got)
	}
	for _, want := range []string{"Asha", "Umoja", "TZS 5,000", "12/09/2026"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing %q", got, want)
		}
	}
}

func TestParseMeetingDate(t *testing.T) {
	for _, in := range []string{"2026-09-22", "22/09/2026", "2026-09-22T10:00:00+03:00"} {
		d, ok := parseMeetingDate(in)
		if !ok || d.Day() != 22 || d.Month() != 9 {
			t.Errorf("parseMeetingDate(%q) = %v, %v", in, d, ok)
		}
	}
	if _, ok := parseMeetingDate("next friday"); ok {
		t.Error("should reject free text")
	}
}
