package mapper

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCellConversionsAndCoordinates(t *testing.T) {
	cell := Cell{Row: 2, Col: 27, Value: " 42 "}
	if value, err := cell.Int64(); err != nil || value != 42 {
		t.Fatalf("Int64() = %d, %v", value, err)
	}
	if CellName(cell.Row, cell.Col) != "AB3" {
		t.Fatalf("CellName() = %q", CellName(cell.Row, cell.Col))
	}
	if _, err := (Cell{Row: 1, Col: 1, Value: "invalid"}).Bool(); err == nil {
		t.Fatal("Bool() error = nil")
	}
	if !reflect.DeepEqual((Cell{Value: "a, b"}).List(","), []string{"a", "b"}) {
		t.Fatal("List() mismatch")
	}
	if value, err := (Cell{Value: "3.5"}).Float64(); err != nil || value != 3.5 {
		t.Fatalf("Float64() = %v, %v", value, err)
	}
	if value, err := (Cell{Value: "2026-08-12"}).Time(""); err != nil || value.Year() != 2026 {
		t.Fatalf("Time() = %v, %v", value, err)
	}
	if (Cell{Value: "x"}).String() != "x" {
		t.Fatal("String() mismatch")
	}
	if (Cell{}).List(",") != nil {
		t.Fatal("empty List() is not nil")
	}
}

func TestDateSystems(t *testing.T) {
	value, err := ExcelSerialTime(61, false)
	if err != nil || value.Format("2006-01-02") != "1900-03-01" {
		t.Fatalf("ExcelSerialTime() = %v, %v", value, err)
	}
	value, err = ParseDate("2026-08-12", "", false)
	if err != nil || value != time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("ParseDate() = %v, %v", value, err)
	}
	if _, err := ExcelSerialTime(1, true); !errors.Is(err, ErrDate1904) {
		t.Fatalf("error = %v", err)
	}
}

func TestSheetNormalizesAndPreservesCoordinates(t *testing.T) {
	sheet := NewSheet("Data", [][]string{{"a"}, {"b", "c"}})
	if len(sheet.Rows[0]) != 2 || sheet.Rows[0][1].Row != 0 || sheet.Rows[1][1].Col != 1 {
		t.Fatalf("sheet = %#v", sheet)
	}
	if !reflect.DeepEqual(sheet.Values(), [][]string{{"a", ""}, {"b", "c"}}) {
		t.Fatalf("values = %#v", sheet.Values())
	}
	if _, err := sheet.Cell(9, 9); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("error = %v", err)
	}
	if _, err := sheet.Rows[0].Cell(9); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("row error = %v", err)
	}
}

func TestSheetOutputToSheet(t *testing.T) {
	output := sheetOutput{Name: "Data"}
	if err := output.PlaceLines(0, Line{"A"}, Line{}, Line{"B", "C"}); err != nil {
		t.Fatal(err)
	}
	sheet := output.Sheet()
	if sheet.Name != "Data" || len(sheet.Rows) != 3 || len(sheet.Rows[0]) != 2 {
		t.Fatalf("sheet = %#v", sheet)
	}
}
