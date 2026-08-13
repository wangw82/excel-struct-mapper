package xlsx

import (
	"fmt"
	"io"
	"strings"

	mapper "github.com/wangw82/excel-struct-mapper"
	"github.com/xuri/excelize/v2"
)

func Write(writer io.Writer, sheet mapper.Sheet, options ...Option) error {
	book, err := NewWorkbook(sheet, options...)
	if err != nil {
		return err
	}
	defer book.Close()
	if err := book.Write(writer); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}
	return nil
}

func WriteFile(path string, sheet mapper.Sheet, options ...Option) error {
	book, err := NewWorkbook(sheet, options...)
	if err != nil {
		return err
	}
	defer book.Close()
	if err := book.SaveAs(path); err != nil {
		return fmt.Errorf("save xlsx %q: %w", path, err)
	}
	return nil
}

func NewWorkbook(sheet mapper.Sheet, options ...Option) (*excelize.File, error) {
	config := configure(options)
	if config.date1904 {
		return nil, mapper.ErrDate1904
	}
	name := config.sheetName
	if name == "" {
		name = sheet.Name
	}
	if name == "" {
		name = "Sheet1"
	}
	book := excelize.NewFile()
	if name != "Sheet1" {
		if err := book.SetSheetName("Sheet1", name); err != nil {
			book.Close()
			return nil, err
		}
	}
	for rowIndex, row := range sheet.Rows {
		for columnIndex, cell := range row {
			coordinate, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err != nil {
				book.Close()
				return nil, err
			}
			value := cell.Value
			if !config.allowFormulas && dangerousFormula(value) {
				value = "'" + value
			}
			if err := book.SetCellStr(name, coordinate, value); err != nil {
				book.Close()
				return nil, fmt.Errorf("set worksheet %q cell %s: %w", name, coordinate, err)
			}
		}
	}
	return book, nil
}

func dangerousFormula(value string) bool {
	value = strings.TrimLeft(value, "\t\r\n ")
	if value == "" {
		return false
	}
	return strings.ContainsRune("=+-@", rune(value[0]))
}
