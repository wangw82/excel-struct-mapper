package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	mapper "github.com/wangw82/excel-struct-mapper"
	"github.com/wangw82/excel-struct-mapper/xlsx"
)

type person struct {
	ID   int    `excel:"header=ID;required=true"`
	Name string `excel:"header=Name;required=true"`
}

type directory struct {
	People []person `excel:"key=people;workflow=all;format=slice"`
}

func main() {
	tempDirectory, err := os.MkdirTemp("", "excel-struct-mapper-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDirectory)
	path := filepath.Join(tempDirectory, "people.xlsx")
	plan, err := mapper.Compile[directory]()
	if err != nil {
		panic(err)
	}
	want := directory{People: []person{{ID: 1, Name: "Ada"}, {ID: 2, Name: "Grace"}}}
	sheet, err := plan.Encode(context.Background(), want)
	if err != nil {
		panic(err)
	}
	if err := xlsx.WriteFile(path, sheet, xlsx.WithSheetName("People")); err != nil {
		panic(err)
	}
	got, err := xlsx.ReadFile(path, xlsx.WithSheetName("People"))
	if err != nil {
		panic(err)
	}
	var decoded directory
	if err := plan.Decode(context.Background(), got, &decoded); err != nil {
		panic(err)
	}
	fmt.Printf("read %d records from %s\n", len(decoded.People), filepath.Base(path))
}
