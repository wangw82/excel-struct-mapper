package xlsx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	mapper "github.com/wangw82/excel-struct-mapper"
	"github.com/xuri/excelize/v2"
)

func TestBufferRoundTripAndFormulaProtection(t *testing.T) {
	want := mapper.NewSheet("Catalog", [][]string{{"Name", "Value"}, {"Chair", "=cmd"}, {"Desk", "+SUM(A1)"}, {"Lamp", "-1+1"}, {"Clock", "@now"}})
	var buffer bytes.Buffer
	if err := Write(&buffer, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(buffer.Bytes()), WithSheetName("Catalog"))
	if err != nil {
		t.Fatal(err)
	}
	values := got.Values()
	for row := 1; row < len(values); row++ {
		if values[row][1][0] != '\'' {
			t.Fatalf("value %q was not protected", values[row][1])
		}
	}
}

func TestFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.xlsx")
	want := mapper.NewSheet("Data", [][]string{{"A", "B"}, {"1", "2"}})
	if err := WriteFile(path, want); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("stat = %v, %v", info, err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Values(), want.Values()) {
		t.Fatalf("got = %#v", got.Values())
	}
}

func TestWorkbookAndFailurePaths(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	if err := book.SetSheetName("Sheet1", "First"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewSheet("Second"); err != nil {
		t.Fatal(err)
	}
	if err := book.SetCellStr("Second", "A1", "selected"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadWorkbook(book, WithSheetName("Second"))
	if err != nil || got.Rows[0][0].Value != "selected" {
		t.Fatalf("got = %#v, %v", got, err)
	}
	if _, err := ReadWorkbook(book, WithSheetName("Missing")); !errors.Is(err, ErrSheetNotFound) {
		t.Fatalf("error = %v", err)
	}
	if _, err := Read(bytes.NewBufferString("broken")); err == nil {
		t.Fatal("invalid input accepted")
	}
	if _, err := ReadWorkbook(book, WithDate1904(true)); !errors.Is(err, mapper.ErrDate1904) {
		t.Fatalf("error = %v", err)
	}
}

func TestFormulasCanBeExplicitlyEnabled(t *testing.T) {
	book, err := NewWorkbook(mapper.NewSheet("Data", [][]string{{"=1+1"}}), WithFormulas(true))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	value, err := book.GetCellValue("Data", "A1")
	if err != nil || value != "=1+1" {
		t.Fatalf("value = %q, %v", value, err)
	}
}
