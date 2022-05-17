package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Posrabi/godb/src"
)

func main() {
	mb := src.NewMemoryBackend()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome")
	for {
		fmt.Print("$ ")
		text, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		text = strings.Replace(text, "\n", "", -1)

		ast, err := src.Parse(text)
		if err != nil {
			panic(err)
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case src.CreateAstKind:
				err = mb.CreateTable(stmt.Create)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")

			case src.InsertAstKind:
				err = mb.Insert(stmt.Insert)
				if err != nil {
					panic(err)
				}
			case src.SelectAstKind:
				results, err := mb.Select(stmt.Select)
				if err != nil {
					panic(err)
				}

				for _, col := range results.Columns {
					fmt.Printf("| %s", col.Name)
				}
				fmt.Println("|")

				for i := 0; i < 20; i++ {
					fmt.Printf("=")
				}
				fmt.Println()

				for _, result := range results.Rows {
					fmt.Printf("|")

					for i, cell := range result {
						typ := results.Columns[i].Type
						s := ""
						switch typ {
						case src.IntType:
							s = fmt.Sprintf("%d", cell.AsInt())
						case src.TextType:
							s = cell.AsText()
						}

						fmt.Printf("%s | ", s)
					}

					fmt.Println()
				}

				fmt.Println("ok")
			}
		}
	}
}
