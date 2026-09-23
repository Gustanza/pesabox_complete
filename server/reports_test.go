package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/xuri/excelize/v2"
)

func sampleSets() []exportSet {
	return []exportSet{
		{
			ds:   findDataset("savings"),
			cols: []string{"Group", "Type", "Amount"},
			rows: []datatype.DataMap{
				{"Group": "Umoja", "Type": "Mandatory Savings", "Amount": 5000.0},
				{"Group": "Amani — B", "Type": "Shares", "Amount": 12500.5},
			},
		},
		{ds: findDataset("members"), cols: []string{"Name", "Gender"}, rows: []datatype.DataMap{{"Name": "Asha", "Gender": "Female"}}},
	}
}

func TestRegistryColumnsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range reportDatasets {
		if seen[d.Key] {
			t.Fatalf("duplicate dataset key %q", d.Key)
		}
		seen[d.Key] = true
		if len(d.Columns) == 0 {
			t.Fatalf("dataset %q has no columns", d.Key)
		}
		for _, c := range d.Columns {
			if _, ok := columnSw[c]; !ok {
				t.Errorf("dataset %q column %q has no Swahili label", d.Key, c)
			}
		}
	}
}

func TestExportXLSXOneSheetPerDataset(t *testing.T) {
	data, err := exportXLSX("sw", sampleSets())
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.GetSheetList(); len(got) != 2 || got[0] != "Akiba" || got[1] != "Wanachama" {
		t.Fatalf("sheets = %v", got)
	}
	if v, _ := f.GetCellValue("Akiba", "A1"); v != "Kikundi" {
		t.Errorf("header = %q, want Kikundi", v)
	}
	if v, _ := f.GetCellValue("Akiba", "B2"); v != "Akiba ya Lazima" {
		t.Errorf("translated type = %q", v)
	}
	if v, _ := f.GetCellValue("Akiba", "C3"); v != "12500.5" {
		t.Errorf("amount = %q, want numeric 12500.5", v)
	}
}

func TestExportCSVSingleAndZip(t *testing.T) {
	sets := sampleSets()
	data, ext, err := exportCSV("en", sets[:1])
	if err != nil || ext != "csv" {
		t.Fatalf("single: ext=%q err=%v", ext, err)
	}
	if !strings.Contains(string(data), "Umoja,Mandatory Savings,5000") {
		t.Errorf("csv body = %q", data)
	}

	data, ext, err = exportCSV("en", sets)
	if err != nil || ext != "zip" {
		t.Fatalf("multi: ext=%q err=%v", ext, err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) != 2 {
		t.Fatalf("zip files=%d err=%v", len(zr.File), err)
	}
}

func TestExportPDF(t *testing.T) {
	for _, lang := range []string{"sw", "en"} {
		data, err := exportPDF(lang, "Umoja", "2026-01-01 - 2026-03-31", sampleSets())
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 {
			t.Fatalf("%s: not a PDF (%d bytes)", lang, len(data))
		}
	}
	// A dataset with no rows must still render (with a "no data" note).
	empty := []exportSet{{ds: findDataset("loans"), cols: loanCols}}
	if _, err := exportPDF("sw", "", "", empty); err != nil {
		t.Fatal(err)
	}
}
