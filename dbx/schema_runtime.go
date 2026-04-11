package dbx

import "github.com/DaiYuANg/arcgo/collectionx"

type schemaSelectItem struct {
	meta ColumnMeta
}

func (s schemaSelectItem) selectItemNode() {}

func (s schemaSelectItem) columnRef() ColumnMeta {
	return s.meta
}

func (s Schema[E]) AllColumns() collectionx.List[SelectItem] {
	return collectionx.MapList(s.def.columns, func(_ int, column ColumnMeta) SelectItem {
		return schemaSelectItem{meta: cloneColumnMeta(column)}
	})
}

func (s Schema[E]) PrimaryColumn() (ColumnMeta, bool) {
	column, ok := collectionx.FindList(s.def.columns, func(_ int, column ColumnMeta) bool {
		return column.PrimaryKey
	})
	if !ok {
		return ColumnMeta{}, false
	}
	return cloneColumnMeta(column), true
}

func (s Schema[E]) ColumnByName(name string) (ColumnMeta, bool) {
	column, ok := s.def.columnByName(name)
	if !ok {
		return ColumnMeta{}, false
	}
	return cloneColumnMeta(column), true
}

type metadataAssignment struct {
	meta  ColumnMeta
	value any
}

type metadataComparisonPredicate struct {
	left  ColumnMeta
	op    ComparisonOperator
	right any
}

type metadataColumnOperand struct {
	meta ColumnMeta
}

func (metadataAssignment) assignmentNode() {}

func (metadataComparisonPredicate) expressionNode() {}
func (metadataComparisonPredicate) predicateNode()  {}
func (metadataColumnOperand) expressionNode()       {}

func (a metadataAssignment) assignmentColumn() ColumnMeta {
	return a.meta
}

func (a metadataAssignment) renderAssignment(state *renderState) error {
	state.writeQuotedIdent(a.meta.Name)
	state.writeString(" = ")
	operand, err := renderOperandValue(state, a.value)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (a metadataAssignment) renderAssignmentValue(state *renderState) error {
	operand, err := renderOperandValue(state, a.value)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (p metadataComparisonPredicate) renderPredicate(state *renderState) error {
	state.renderColumn(p.left)
	if p.op == OpIs || p.op == OpIsNot {
		state.writeByte(' ')
		state.writeString(string(p.op))
		state.writeString(" NULL")
		return nil
	}

	operand, err := renderOperandValue(state, p.right)
	if err != nil {
		return err
	}
	state.writeByte(' ')
	state.writeString(string(p.op))
	state.writeByte(' ')
	state.writeString(operand)
	return nil
}

func (o metadataColumnOperand) renderOperand(state *renderState) (string, error) {
	var builder renderBuffer
	table := o.meta.Table
	if o.meta.Alias != "" {
		table = o.meta.Alias
	}
	builder.writeString(state.dialect.QuoteIdent(table))
	builder.writeByte('.')
	builder.writeString(state.dialect.QuoteIdent(o.meta.Name))
	return builder.String(), builder.Err("render metadata column operand")
}
