package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Posrabi/godb/src"
	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
)

func doSelect(mb src.Backend, slct *src.SelectStatement) error {
	results, err := mb.Select(slct)
	if err != nil {
		return err
	}

	if len(results.Rows) == 0 {
		log.Println("(no results)")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{}
	for _, col := range results.Columns {
		header = append(header, col.Name)
	}
	table.SetHeader(header)
	table.SetAutoFormatHeaders(false)

	rows := [][]string{}
	for _, result := range results.Rows {
		row := []string{}
		for i, cell := range result {
			typ := results.Columns[i].Type
			s := ""

			switch typ {
			case src.IntType:
				s = fmt.Sprintf("%d", cell.AsInt())
			case src.TextType:
				s = cell.AsText()
			case src.BoolType:
				s = "true"
				if !cell.AsBool() {
					s = "false"
				}
			}

			row = append(row, s)
		}
		rows = append(rows, row)
	}

	table.SetBorder(false)
	table.AppendBulk(rows)
	table.Render()

	if len(rows) == 1 {
		log.Println("(1 result)")
	} else {
		log.Printf("(%d results)", len(rows))
	}

	return nil
}

func main() {
	mb := src.NewMemoryBackend()

	l, err := readline.NewEx(&readline.Config{
		Prompt:          "# ",
		HistoryFile:     "/tmp/gosql.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	fmt.Println("Welcome")

repl:
	for {
		fmt.Print("$ ")
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue repl
			}
		} else if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("Error while reading line:", err)
			continue repl
		}

		if line == "clear" {
			fmt.Print("\033[H\033[2J")
			continue repl
		} else if line == "exit" {
			log.Println("Exiting...")
			break
		}

		ast, err := src.Parse(line)
		if err != nil {
			log.Println("Error while parsing:", err)
			continue repl
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case src.CreateAstKind:
				err = mb.CreateTable(stmt.Create)
				if err != nil {
					log.Println("Error creating table:", err)
					continue repl
				}

			case src.InsertAstKind:
				err = mb.Insert(stmt.Insert)
				if err != nil {
					log.Println("Error inserting value:", err)
					continue repl
				}

			case src.SelectAstKind:
				err := doSelect(mb, stmt.Select)
				if err != nil {
					log.Println("Error selecting values:", err)
					continue repl
				}
			}
			fmt.Println("ok")
		}
	}
}
