package mapper

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	KindConfiguration ErrorKind = "configuration"
	KindConversion    ErrorKind = "conversion"
	KindValidation    ErrorKind = "validation"
	KindSplit         ErrorKind = "split"
	KindUnconsumedRow ErrorKind = "unconsumed_row"
	KindCanceled      ErrorKind = "canceled"
)

var (
	ErrInvalidModel        = errors.New("model must be a struct")
	ErrInvalidTarget       = errors.New("target must be a non-nil pointer compatible with the plan")
	ErrInvalidSource       = errors.New("source is incompatible with the plan")
	ErrInvalidTag          = errors.New("invalid excel tag")
	ErrDuplicateKey        = errors.New("duplicate block key")
	ErrUnknownWorkflow     = errors.New("unknown block workflow")
	ErrUnknownBlockFormat  = errors.New("unknown block format")
	ErrUnknownValueCodec   = errors.New("unknown value codec")
	ErrUnknownBlockCodec   = errors.New("unknown block codec")
	ErrUnknownTagHandler   = errors.New("unknown tag handler")
	ErrUnknownTag          = errors.New("unknown registered tag")
	ErrInvalidBlockBinding = errors.New("invalid block binding")
	ErrInvalidRegistry     = errors.New("invalid registry")
	ErrInvalidBlock        = errors.New("invalid block returned by workflow")
	ErrLayoutConflict      = errors.New("worksheet block layout conflict")
	ErrSheetEmpty          = errors.New("sheet is empty")
	ErrOutOfBounds         = errors.New("cell is out of bounds")
	ErrDate1904            = errors.New("Excel 1904 date system is not supported")
)

type Error struct {
	Kind  ErrorKind
	Sheet string
	Block string
	Field string
	Row   int
	Col   int
	Cell  string
	Cause error
}

func (e *Error) Error() string {
	location := ""
	if e.Cell != "" {
		location = " at " + e.Cell
	}
	if e.Sheet != "" {
		location += " in sheet " + fmt.Sprintf("%q", e.Sheet)
	}
	field := ""
	if e.Field != "" {
		field = " field " + e.Field
	}
	block := ""
	if e.Block != "" {
		block = " block " + e.Block
	}
	return fmt.Sprintf("%s%s%s%s: %v", e.Kind, block, field, location, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

type ValidationIssue struct {
	Field string
	Rule  string
	Value string
	Cause error
}

func (i ValidationIssue) Error() string {
	if i.Cause != nil {
		return fmt.Sprintf("field %s failed %s: %v", i.Field, i.Rule, i.Cause)
	}
	return fmt.Sprintf("field %s failed %s", i.Field, i.Rule)
}

func (i ValidationIssue) Unwrap() error { return i.Cause }

func locatedError(kind ErrorKind, sheet, block, field string, row, col int, cause error) error {
	cell := ""
	if row >= 0 && col >= 0 {
		cell = CellName(row, col)
	}
	return &Error{Kind: kind, Sheet: sheet, Block: block, Field: field, Row: row + 1, Col: col + 1, Cell: cell, Cause: cause}
}
