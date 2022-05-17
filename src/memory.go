package src

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type memoryCell []byte

func (mc memoryCell) AsInt() int32 {
	var i int32
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		panic(err)
	}

	return i
}

func (mc memoryCell) AsText() string {
	return string(mc)
}

type table struct {
	columns     []string
	columnTypes []columnType
	rows        [][]memoryCell
}

type MemoryBackend struct {
	tables map[string]*table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}

func (mb *MemoryBackend) CreateTable(crt *createTableStatement) error {
	t := table{}
	mb.tables[crt.name.Value] = &t
	if crt.cols == nil {
		return nil
	}

	for _, col := range *crt.cols {
		t.columns = append(t.columns, col.name.Value)

		var dt columnType
		switch col.dataType.Value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		default:
			return InvalidDatatype
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	return nil
}

func (mb *MemoryBackend) Insert(inst *insertStatement) error {
	table, ok := mb.tables[inst.table.Value]
	if !ok {
		return TableDoesNotExists
	}

	if inst.values == nil {
		return nil
	}

	row := []memoryCell{}

	if len(*inst.values) != len(table.columns) {
		return MissingValues
	}

	for _, value := range *inst.values {
		if value.kind != literal {
			fmt.Println("Skipping non-literal")
			continue
		}

		row = append(row, mb.tokenToCell(value.literal))
	}

	table.rows = append(table.rows, row)
	return nil
}

func (mb *MemoryBackend) tokenToCell(t *token) memoryCell {
	if t.Kind == NumericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.Value)
		if err != nil {
			panic(err)
		}

		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			panic(err)
		}
		return memoryCell(buf.Bytes())
	}

	if t.Kind == StringKind {
		return memoryCell(t.Value)
	}

	return nil
}

func (mb *MemoryBackend) Select(slct *selectStatement) (*Results, error) {
	table, ok := mb.tables[slct.from.Value]
	if !ok {
		return nil, TableDoesNotExists
	}

	results := [][]Cell{}
	columns := []struct {
		Type columnType
		Name string
	}{}

	for i, row := range table.rows {
		result := []Cell{}
		isFirstRow := i == 0

		for _, exp := range slct.item {
			if exp.kind != literal {
				fmt.Println("Skipping non-literal expression")
				continue
			}

			lit := exp.literal
			if lit.Kind == IdentifierKind {
				found := false
				for i, tableCol := range table.columns {
					if tableCol == lit.Value {
						if isFirstRow {
							columns = append(columns, struct {
								Type columnType
								Name string
							}{Type: table.columnTypes[i], Name: lit.Value})
						}

						result = append(result, row[i])
						found = true
						break
					}
				}

				if !found {
					return nil, ColumnDoesNotExist
				}

				continue
			}
		}

		results = append(results, result)
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}
