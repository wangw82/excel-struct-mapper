package main

import (
	"context"
	"fmt"

	mapper "github.com/wangw82/excel-struct-mapper"
)

type product struct {
	ID   int    `excel:"header=ID;required=true"`
	Name string `excel:"header=Name;required=true"`
}

type catalog struct {
	Products []product `excel:"key=products;workflow=all;format=slice"`
}

func main() {
	plan, err := mapper.Compile[catalog]()
	if err != nil {
		panic(err)
	}
	sheet := mapper.NewSheet("Catalog", [][]string{
		{"ID", "Name"},
		{"7", "Example"},
	})
	var value catalog
	if err := plan.Decode(context.Background(), sheet, &value); err != nil {
		panic(err)
	}
	fmt.Printf("%d %s\n", value.Products[0].ID, value.Products[0].Name)
}
