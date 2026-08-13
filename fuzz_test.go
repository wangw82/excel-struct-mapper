package mapper

import "testing"

func FuzzParseOptions(f *testing.F) {
	f.Add("header=ID;required=true")
	f.Add(`header=A\;B`)
	f.Fuzz(func(t *testing.T, value string) { _, _ = parseOptions(value, fieldOptions) })
}

func FuzzExcelSerial(f *testing.F) {
	f.Add(1.0)
	f.Add(45200.5)
	f.Fuzz(func(t *testing.T, value float64) { _, _ = ExcelSerialTime(value, false) })
}
