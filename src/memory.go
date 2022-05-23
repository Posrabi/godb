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

func (mc memoryCell) AsBool() bool {
	return len(mc) != 0
}

func (mc memoryCell) equals(b memoryCell) bool {
	if mc == nil || b == nil {
		return mc == nil && b == nil
	}

	return bytes.Compare(mc, b) == 0
}

type MemoryBackend struct {
	tables map[string]*table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}

func (mb *MemoryBackend) CreateTable(crt *CreateTableStatement) error {
	t := newTable()
	mb.tables[crt.name.value] = t
	if crt.cols == nil {
		return nil
	}

	for _, col := range *crt.cols {
		t.columns = append(t.columns, col.name.value)

		var dt columnType
		switch col.dataType.value {
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

func (mb *MemoryBackend) Insert(inst *InsertStatement) error {
	table, ok := mb.tables[inst.table.value]
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

		emptyTable := newTable()
		value, _, _, err := emptyTable.evaluateCell(0, *value)
		if err != nil {
			return err
		}

		row = append(row, value)
	}

	table.rows = append(table.rows, row)
	return nil
}

func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	table := newTable()

	if slct.from != nil {
		var ok bool
		table, ok = mb.tables[slct.from.value]
		if !ok {
			return nil, TableDoesNotExists
		}
	}

	if slct.item == nil || len(*slct.item) == 0 {
		return &Results{}, nil
	}

	results := [][]Cell{}
	columns := []struct {
		Type columnType
		Name string
	}{}

	for i := range table.rows {
		result := []Cell{}
		isFirstRow := len(results) == 0

		if slct.where != nil {
			val, _, _, err := table.evaluateCell(uint(i), *slct.where)
			if err != nil {
				return nil, err
			}

			if !val.AsBool() {
				continue
			}
		}

		for _, col := range *slct.item {
			if col.asterisk {
				fmt.Println("Skipping asterisk.")
				continue
			}

			value, colName, colType, err := table.evaluateCell(uint(i), *col.exp)
			if err != nil {
				return nil, err
			}

			if isFirstRow {
				columns = append(columns, struct {
					Type columnType
					Name string
				}{colType, colName})
			}

			result = append(result, value)
		}

		results = append(results, result)
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (mb *MemoryBackend) tokenToCell(t *token) memoryCell {
	if t.kind == NumericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.value)
		if err != nil {
			panic(err)
		}

		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			panic(err)
		}
		return memoryCell(buf.Bytes())
	}

	if t.kind == StringKind {
		return memoryCell(t.value)
	}

	return nil
}

type table struct {
	columns     []string
	columnTypes []columnType
	rows        [][]memoryCell
}

func newTable() *table {
	return &table{
		columns:     nil,
		columnTypes: nil,
		rows:        nil,
	}
}

func (t *table) evaluateCell(rowIndex uint, exp expression) (memoryCell, string, columnType, error) {
	switch exp.kind {
	case literal:
		return t.evaluateLiteralCell(rowIndex, exp)
	case binaryKind:
		return t.evaluateBinaryCell(rowIndex, exp)
	default:
		return nil, "", 0, InvalidCell
	}
}

func (t *table) evaluateLiteralCell(rowIndex uint, exp expression) (memoryCell, string, columnType, error) {
	if exp.kind != literal {
		return nil, "", 0, InvalidCell
	}

	lit := exp.literal
	if lit.kind == IdentifierKind {
		for i, tableCol := range t.columns {
			if tableCol == lit.value {
				return t.rows[rowIndex][i], tableCol, t.columnTypes[i], nil
			}
		}

		return nil, "", 0, ColumnDoesNotExist
	}

	columnType := IntType
	if lit.kind == StringKind {
		columnType = TextType
	} else if lit.kind == BoolKind {
		columnType = BoolType
	}

	return lit.literalToMemoryCell(), "?column?", columnType, nil
}

func (t *table) evaluateBinaryCell(rowIndex uint, exp expression) (memoryCell, string, columnType, error) {
	if exp.kind != binaryKind {
		return nil, "", 0, InvalidCell
	}

	bexp := exp.binary

	left, _, leftType, err := t.evaluateCell(rowIndex, bexp.a)
	if err != nil {
		return nil, "", 0, err
	}

	right, _, rightType, err := t.evaluateCell(rowIndex, bexp.b)
	if err != nil {
		return nil, "", 0, err
	}

	columnName := fmt.Sprintf("%s %s %s", bexp.a.literal.value, bexp.op.value, bexp.b.literal.value)

	switch bexp.op.kind {
	case SymbolKind:
		switch symbol(bexp.op.value) {
		case Equal:
			eq := left.equals(right)
			if leftType == TextType && rightType == TextType && eq {
				return trueMemoryCell, columnName, BoolType, nil
			}

			if leftType == IntType && rightType == IntType && eq {
				return trueMemoryCell, columnName, BoolType, nil
			}

			if leftType == BoolType && rightType == BoolType && eq {
				return trueMemoryCell, columnName, BoolType, nil
			}
			return falseMemoryCell, columnName, BoolType, nil
		case XEqual:
			if leftType != rightType || !left.equals(right) {
				return trueMemoryCell, columnName, BoolType, nil
			}

			return falseMemoryCell, columnName, BoolType, nil
		case Concat:
			if leftType != TextType || rightType != TextType {
				return nil, "", 0, InvalidOperands
			}

			lit := &token{kind: StringKind, value: left.AsText() + right.AsText()}
			return lit.literalToMemoryCell(), columnName, TextType, nil
		case Plus:
			if leftType != IntType || rightType != IntType {
				return nil, "", 0, InvalidOperands
			}

			lit := &token{kind: NumericKind, value: strconv.Itoa(int(left.AsInt() + right.AsInt()))}
			return lit.literalToMemoryCell(), columnName, IntType, nil
		default:
			// TODO
			break
		}
	case KeywordKind:
		switch keyword(bexp.op.value) {
		case And:
			if leftType != BoolType || rightType != BoolType {
				return nil, "", 0, InvalidOperands
			}

			res := falseMemoryCell
			if left.AsBool() && right.AsBool() {
				res = trueMemoryCell
			}

			return res, columnName, BoolType, nil
		case Or:
			if leftType != BoolType || rightType != BoolType {
				return nil, "", 0, InvalidOperands
			}

			res := falseMemoryCell
			if left.AsBool() || right.AsBool() {
				res = trueMemoryCell
			}

			return res, columnName, BoolType, nil
		default:
			//TODO
			break
		}
	}

	return nil, "", 0, InvalidCell
}
