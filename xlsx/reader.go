package xlsx

import (
	"errors"
	"fmt"
	"io"

	mapper "github.com/wangw82/excel-struct-mapper"
	"github.com/xuri/excelize/v2"
)

var ErrSheetNotFound = errors.New("worksheet not found")

func Read(reader io.Reader, options ...Option) (mapper.Sheet, error) {
	book, err := excelize.OpenReader(reader)
	if err != nil {
		return mapper.Sheet{}, fmt.Errorf("open xlsx: %w", err)
	}
	defer book.Close()
	return ReadWorkbook(book, options...)
}

func ReadFile(path string, options ...Option) (mapper.Sheet, error) {
	book, err := excelize.OpenFile(path)
	if err != nil {
		return mapper.Sheet{}, fmt.Errorf("open xlsx %q: %w", path, err)
	}
	defer book.Close()
	return ReadWorkbook(book, options...)
}

func ReadWorkbook(book *excelize.File, options ...Option) (mapper.Sheet, error) {
	if book == nil {
		return mapper.Sheet{}, fmt.Errorf("nil workbook")
	}
	config := configure(options)
	if config.date1904 {
		return mapper.Sheet{}, mapper.ErrDate1904
	}
	name, err := resolveSheet(book, config.sheetName)
	if err != nil {
		return mapper.Sheet{}, err
	}
	rows, err := book.GetRows(name, excelize.Options{RawCellValue: true})
	if err != nil {
		return mapper.Sheet{}, fmt.Errorf("read worksheet %q: %w", name, err)
	}
	return mapper.NewSheet(name, rows), nil
}

func resolveSheet(book *excelize.File, requested string) (string, error) {
	if requested != "" {
		index, err := book.GetSheetIndex(requested)
		if err != nil || index < 0 {
			return "", fmt.Errorf("%w: %q", ErrSheetNotFound, requested)
		}
		return requested, nil
	}
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return "", ErrSheetNotFound
	}
	return sheets[0], nil
}
