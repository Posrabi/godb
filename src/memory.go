package src

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"

	"github.com/petar/GoLLRB/llrb"
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
	if _, ok := mb.tables[crt.name.value]; ok {
		return TableAlreadyExists
	}

	t := newTable()
	t.name = crt.name.value
	mb.tables[t.name] = t
	if crt.cols == nil {
		return nil
	}

	var primaryKey *expression = nil
	for _, col := range *crt.cols {
		t.columns = append(t.columns, col.name.value)

		var dt columnType
		switch col.dataType.value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		default:
			delete(mb.tables, t.name)
			return InvalidDatatype
		}

		if col.primaryKey {
			if primaryKey != nil {
				delete(mb.tables, t.name)
				return PrimaryKeyAlreadyExists
			}

			primaryKey = &expression{
				literal: &col.name,
				kind:    literal,
			}
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	if primaryKey != nil {
		err := mb.CreateIndex(&CreateIndexStatement{
			table:      crt.name,
			name:       token{value: t.name + "_pkey"},
			unique:     true,
			primaryKey: true,
			exp:        *primaryKey,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (mb *MemoryBackend) CreateIndex(ci *CreateIndexStatement) error {
	table, ok := mb.tables[ci.table.value]
	if !ok {
		return TableDoesNotExists
	}

	for _, index := range table.indexes {
		if index.name == ci.name.value {
			return IndexAlreadyExists
		}
	}

	index := &index{
		exp:        ci.exp,
		unique:     ci.unique,
		primaryKey: ci.primaryKey,
		name:       ci.name.value,
		tree:       llrb.New(),
		typ:        "rbtree",
	}
	table.indexes = append(table.indexes, index)
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
	columns := []ResultsColumn{}

	for _, iAndE := range table.getApplicableIndexes(slct.where) {
		index := iAndE.i
		exp := iAndE.e
		table = index.newTableFromSubset(table, exp)
	}

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
			value, colName, colType, err := table.evaluateCell(uint(i), *col.exp)
			if err != nil {
				return nil, err
			}

			if isFirstRow {
				columns = append(columns, ResultsColumn{colType, colName})
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
	indexes     []*index
	name        string
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

func (t *table) getApplicableIndexes(where *expression) []indexAndExpression {
	var linearizeExpressions func(where *expression, exps []expression) []expression

	linearizeExpressions = func(where *expression, exps []expression) []expression {
		if where == nil || where.kind != binaryKind {
			return exps
		}

		if where.binary.op.value == string(Or) {
			return exps
		}

		if where.binary.op.value == string(And) {
			exps := linearizeExpressions(&where.binary.a, exps)
			return linearizeExpressions(&where.binary.b, exps)
		}

		return append(exps, *where)
	}

	exps := linearizeExpressions(where, []expression{})

	iAndE := []indexAndExpression{}
	for _, exp := range exps {
		for _, index := range t.indexes {
			if index.applicableValue(exp) != nil {
				iAndE = append(iAndE, indexAndExpression{
					i: index,
					e: exp,
				})
			}
		}
	}

	return iAndE
}

// Implements llrb.Item interface
type treeItem struct {
	value memoryCell
	index uint
}

func (ti treeItem) Less(than llrb.Item) bool {
	return bytes.Compare(ti.value, than.(treeItem).value) < 0
}

type index struct {
	name       string
	exp        expression
	unique     bool
	primaryKey bool
	tree       *llrb.LLRB
	typ        string
}

func (i *index) addRow(t *table, rowIndex uint) error {
	indexValue, _, _, err := t.evaluateCell(rowIndex, i.exp)
	if err != nil {
		return err
	}

	if indexValue == nil {
		return ViolatesNonNullConstraint
	}

	if i.unique && i.tree.Has(treeItem{value: indexValue}) {
		return ViolatesUniqueConstraint
	}

	i.tree.InsertNoReplace(treeItem{
		value: indexValue,
		index: rowIndex,
	})
	return nil
}

// Support matching for =, <>, >, <, >=, or <=
// One of the operands is an identifier that match the index
// The other is a literal value
func (i *index) applicableValue(exp expression) *expression {
	if exp.kind != binaryKind {
		return nil
	}

	be := exp.binary
	// Find the column and the value in the boolean expression
	columnExp := be.a
	valueExp := be.b
	if columnExp.generateCode() != i.exp.generateCode() {
		return nil
	}

	supportedChecks := []symbol{Equal, XEqual, Greater, GreaterOrEqual, Less, LessOrEqual}
	supported := false
	for _, sym := range supportedChecks {
		if be.op.value == string(sym) {
			supported = true
			break
		}
	}
	if !supported {
		return nil
	}

	if valueExp.kind != literal {
		fmt.Println("only index checks on literals supported")
		return nil
	}

	return &valueExp
}

func (i *index) newTableFromSubset(t *table, exp expression) *table {
	valueExp := i.applicableValue(exp)
	if valueExp == nil {
		return t
	}

	value, _, _, err := newTable().evaluateCell(0, *valueExp)
	if err != nil {
		log.Println(err)
		return t
	}

	tiValue := treeItem{value: value}

	fmt.Println(symbol(exp.binary.op.value), symbol(exp.binary.op.value) == Equal)
	indexes := []uint{}
	switch symbol(exp.binary.op.value) {
	case Equal:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)

			fmt.Println(ti.value, value)
			if !bytes.Equal(ti.value, value) {
				return false
			}

			indexes = append(indexes, ti.index)
			return true
		})
	case XEqual:
		i.tree.AscendGreaterOrEqual(llrb.Int(-1), func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Equal(ti.value, value) {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case Less:
		i.tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) < 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case LessOrEqual:
		i.tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) <= 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case Greater:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) > 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case GreaterOrEqual:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) >= 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	}

	newT := newTable()
	newT.columns = t.columns
	newT.columnTypes = t.columnTypes
	newT.indexes = t.indexes
	newT.rows = [][]memoryCell{}

	for _, index := range indexes {
		newT.rows = append(newT.rows, t.rows[index])
	}

	return newT
}

type indexAndExpression struct {
	i *index
	e expression
}
