package dbx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type predicateRenderer interface {
	renderPredicate(*renderState) error
}

type assignmentRenderer interface {
	renderAssignment(*renderState) error
}

type insertAssignmentRenderer interface {
	assignmentRenderer
	renderAssignmentValue(*renderState) error
	assignmentColumn() ColumnMeta
}

type orderRenderer interface {
	renderOrder(*renderState) error
}

type operandRenderer interface {
	renderOperand(*renderState) (string, error)
}

type selectItemRenderer interface {
	renderSelectItem(*renderState) error
}

type renderState struct {
	dialect  dialect.Dialect
	buf      strings.Builder
	args     []any
	writeErr error
}

func (s *renderState) writeString(text string) {
	if s.writeErr != nil {
		return
	}

	_, s.writeErr = s.buf.WriteString(text)
}

func (s *renderState) writeByte(value byte) {
	if s.writeErr != nil {
		return
	}

	s.writeErr = s.buf.WriteByte(value)
}

func (s *renderState) err() error {
	return wrapDBError("write rendered SQL", s.writeErr)
}

type renderBuffer struct {
	buf strings.Builder
	err error
}

func (b *renderBuffer) writeString(text string) {
	if b.err != nil {
		return
	}

	_, b.err = b.buf.WriteString(text)
}

func (b *renderBuffer) writeByte(value byte) {
	if b.err != nil {
		return
	}

	b.err = b.buf.WriteByte(value)
}

func (b *renderBuffer) String() string {
	return b.buf.String()
}

func (b *renderBuffer) Err(op string) error {
	return wrapDBError(op, b.err)
}

func (s *renderState) bind(value any) string {
	s.args = append(s.args, value)
	return s.dialect.BindVar(len(s.args))
}

func (s *renderState) writeQuotedIdent(name string) {
	s.writeString(s.dialect.QuoteIdent(name))
}

func (s *renderState) writeQualifiedIdent(table, column string) {
	if table != "" {
		s.writeQuotedIdent(table)
		s.writeByte('.')
	}
	s.writeQuotedIdent(column)
}

func (s *renderState) renderColumn(meta ColumnMeta) {
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	s.writeQualifiedIdent(table, meta.Name)
}

func (s *renderState) renderTable(table Table) {
	s.writeQuotedIdent(table.Name())
	if alias := table.Alias(); alias != "" && alias != table.Name() {
		s.writeString(" AS ")
		s.writeQuotedIdent(alias)
	}
}

func (s *renderState) BoundQuery() BoundQuery {
	args := make([]any, len(s.args))
	copy(args, s.args)
	return BoundQuery{SQL: s.buf.String(), Args: args}
}

func (q *SelectQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: select query is nil")
	}
	if q.FromItem.Name() == "" {
		return BoundQuery{}, errors.New("dbx: select query requires FROM")
	}
	if len(q.Items) == 0 {
		return BoundQuery{}, errors.New("dbx: select query requires at least one item")
	}

	state := &renderState{dialect: d, args: make([]any, 0, 8)}
	if err := renderSelectStatement(state, q); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	bound := state.BoundQuery()
	if q.LimitN != nil && *q.LimitN > 0 {
		bound.CapacityHint = *q.LimitN
	}
	return bound, nil
}

func renderSelectStatement(state *renderState, q *SelectQuery) error {
	if err := renderCTEs(state, q.CTEs); err != nil {
		return err
	}
	return renderSelectSet(state, q)
}

func renderSelectSet(state *renderState, q *SelectQuery) error {
	if len(q.Unions) == 0 {
		return renderSelectQuery(state, q)
	}

	if err := renderSelectQueryWithoutTail(state, q); err != nil {
		return err
	}
	for _, union := range q.Unions {
		if union.Query == nil {
			return errors.New("dbx: union query is nil")
		}
		if union.All {
			state.writeString(" UNION ALL ")
		} else {
			state.writeString(" UNION ")
		}
		if err := renderUnionQuery(state, union.Query); err != nil {
			return err
		}
	}
	return renderSelectTail(state, q)
}

func renderCTEs(state *renderState, ctes []CTE) error {
	if len(ctes) == 0 {
		return nil
	}
	state.writeString("WITH ")
	for index, cte := range ctes {
		if strings.TrimSpace(cte.Name) == "" {
			return errors.New("dbx: cte name cannot be empty")
		}
		if cte.Query == nil {
			return fmt.Errorf("dbx: cte %s requires query", cte.Name)
		}
		if index > 0 {
			state.writeString(", ")
		}
		state.writeQuotedIdent(strings.TrimSpace(cte.Name))
		state.writeString(" AS (")
		if err := renderSelectStatement(state, cte.Query); err != nil {
			return err
		}
		state.writeByte(')')
	}
	state.writeByte(' ')
	return nil
}

func renderUnionQuery(state *renderState, q *SelectQuery) error {
	if len(q.CTEs) > 0 || len(q.Unions) > 0 || len(q.Orders) > 0 || q.LimitN != nil || q.OffsetN != nil {
		state.writeByte('(')
		if err := renderSelectStatement(state, q); err != nil {
			return err
		}
		state.writeByte(')')
		return nil
	}
	return renderSelectQueryWithoutTail(state, q)
}

func renderSelectQuery(state *renderState, q *SelectQuery) error {
	if err := renderSelectQueryWithoutTail(state, q); err != nil {
		return err
	}
	return renderSelectTail(state, q)
}

func renderSelectQueryWithoutTail(state *renderState, q *SelectQuery) error {
	if err := renderSelectDistinct(state, q); err != nil {
		return err
	}
	if err := renderSelectItems(state, q); err != nil {
		return err
	}
	if err := renderSelectFrom(state, q); err != nil {
		return err
	}
	if err := renderSelectJoins(state, q); err != nil {
		return err
	}
	if err := renderSelectWhere(state, q); err != nil {
		return err
	}
	if err := renderSelectGroupBy(state, q); err != nil {
		return err
	}
	if err := renderSelectHaving(state, q); err != nil {
		return err
	}
	return nil
}

func renderSelectDistinct(state *renderState, q *SelectQuery) error {
	state.writeString("SELECT ")
	if q.Distinct {
		state.writeString("DISTINCT ")
	}
	return nil
}

func renderSelectItems(state *renderState, q *SelectQuery) error {
	for i, item := range q.Items {
		if i > 0 {
			state.writeString(", ")
		}
		if err := renderSelectItem(state, item); err != nil {
			return err
		}
	}
	return nil
}

func renderSelectFrom(state *renderState, q *SelectQuery) error {
	state.writeString(" FROM ")
	state.renderTable(q.FromItem)
	return nil
}

func renderSelectJoins(state *renderState, q *SelectQuery) error {
	for _, join := range q.Joins {
		state.writeByte(' ')
		state.writeString(string(join.Type))
		state.writeString(" JOIN ")
		state.renderTable(join.Table)
		if join.Predicate != nil {
			state.writeString(" ON ")
			if err := renderPredicate(state, join.Predicate); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderSelectWhere(state *renderState, q *SelectQuery) error {
	if q.WhereExp != nil {
		state.writeString(" WHERE ")
		if err := renderPredicate(state, q.WhereExp); err != nil {
			return err
		}
	}
	return nil
}

func renderSelectGroupBy(state *renderState, q *SelectQuery) error {
	if len(q.Groups) > 0 {
		state.writeString(" GROUP BY ")
		for i, group := range q.Groups {
			if i > 0 {
				state.writeString(", ")
			}
			operand, err := renderOperandValue(state, group)
			if err != nil {
				return err
			}
			state.writeString(operand)
		}
	}
	return nil
}

func renderSelectHaving(state *renderState, q *SelectQuery) error {
	if q.HavingExp != nil {
		state.writeString(" HAVING ")
		if err := renderPredicate(state, q.HavingExp); err != nil {
			return err
		}
	}
	return nil
}

func renderSelectTail(state *renderState, q *SelectQuery) error {
	if len(q.Orders) > 0 {
		state.writeString(" ORDER BY ")
		for i, order := range q.Orders {
			if i > 0 {
				state.writeString(", ")
			}
			if err := renderOrder(state, order); err != nil {
				return err
			}
		}
	}
	clause, err := state.dialect.RenderLimitOffset(q.LimitN, q.OffsetN)
	if err != nil {
		return fmt.Errorf("dbx: render limit offset: %w", err)
	}
	if clause != "" {
		state.writeByte(' ')
		state.writeString(clause)
	}
	return nil
}

func (q *InsertQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: insert query is nil")
	}
	if q.Into.Name() == "" {
		return BoundQuery{}, errors.New("dbx: insert query requires target table")
	}
	rows := normalizedInsertRows(q)
	if len(rows) == 0 && q.Source == nil {
		return BoundQuery{}, errors.New("dbx: insert query requires values or source query")
	}
	if len(rows) > 0 && q.Source != nil {
		return BoundQuery{}, errors.New("dbx: insert query cannot combine values and source query")
	}
	if q.Source != nil && len(q.TargetColumns) == 0 {
		return BoundQuery{}, errors.New("dbx: insert-select requires target columns")
	}

	state := &renderState{dialect: d, args: make([]any, 0, len(rows)*4)}
	features := dialectFeatures(d)
	if features.InsertIgnoreForUpsertNothing && q.Upsert != nil && q.Upsert.DoNothing {
		state.writeString("INSERT IGNORE INTO ")
	} else {
		state.writeString("INSERT INTO ")
	}
	if err := renderInsertBody(state, q, rows); err != nil {
		return BoundQuery{}, err
	}
	if err := renderUpsert(state, q); err != nil {
		return BoundQuery{}, err
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func renderInsertBody(state *renderState, q *InsertQuery, rows [][]Assignment) error {
	state.renderTable(q.Into)
	columns, err := resolveInsertColumns(q, rows)
	if err != nil {
		return err
	}
	if len(columns) > 0 {
		state.writeString(" (")
		for i := range columns {
			column := &columns[i]
			if i > 0 {
				state.writeString(", ")
			}
			state.writeQuotedIdent(column.Name)
		}
		state.writeByte(')')
	}
	if q.Source != nil {
		state.writeByte(' ')
		return renderSelectQuery(state, q.Source)
	}
	orderedRows, err := orderInsertRows(columns, rows)
	if err != nil {
		return err
	}
	state.writeString(" VALUES ")
	for rowIndex, row := range orderedRows {
		if rowIndex > 0 {
			state.writeString(", ")
		}
		state.writeByte('(')
		for colIndex, assignment := range row {
			renderer, ok := assignment.(insertAssignmentRenderer)
			if !ok {
				return fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
			}
			if colIndex > 0 {
				state.writeString(", ")
			}
			if err := renderer.renderAssignmentValue(state); err != nil {
				return err
			}
		}
		state.writeByte(')')
	}
	return nil
}

func (q *UpdateQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: update query is nil")
	}
	if q.Table.Name() == "" {
		return BoundQuery{}, errors.New("dbx: update query requires target table")
	}
	if len(q.Assignments) == 0 {
		return BoundQuery{}, errors.New("dbx: update query requires assignments")
	}

	state := &renderState{dialect: d, args: make([]any, 0, len(q.Assignments))}
	state.writeString("UPDATE ")
	state.renderTable(q.Table)
	state.writeString(" SET ")
	for i, assignment := range q.Assignments {
		if i > 0 {
			state.writeString(", ")
		}
		if err := renderAssignment(state, assignment); err != nil {
			return BoundQuery{}, err
		}
	}
	if q.WhereExp != nil {
		state.writeString(" WHERE ")
		if err := renderPredicate(state, q.WhereExp); err != nil {
			return BoundQuery{}, err
		}
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func (q *DeleteQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: delete query is nil")
	}
	if q.From.Name() == "" {
		return BoundQuery{}, errors.New("dbx: delete query requires target table")
	}

	state := &renderState{dialect: d, args: make([]any, 0, 4)}
	state.writeString("DELETE FROM ")
	state.renderTable(q.From)
	if q.WhereExp != nil {
		state.writeString(" WHERE ")
		if err := renderPredicate(state, q.WhereExp); err != nil {
			return BoundQuery{}, err
		}
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func renderSelectItem(state *renderState, item SelectItem) error {
	if renderer, ok := item.(selectItemRenderer); ok {
		return renderer.renderSelectItem(state)
	}
	column, ok := item.(columnAccessor)
	if !ok {
		return fmt.Errorf("dbx: unsupported select item %T", item)
	}
	state.renderColumn(column.columnRef())
	return nil
}

func renderPredicate(state *renderState, predicate Predicate) error {
	renderer, ok := predicate.(predicateRenderer)
	if !ok {
		return fmt.Errorf("dbx: unsupported predicate %T", predicate)
	}
	return renderer.renderPredicate(state)
}

func renderAssignment(state *renderState, assignment Assignment) error {
	renderer, ok := assignment.(assignmentRenderer)
	if !ok {
		return fmt.Errorf("dbx: unsupported assignment %T", assignment)
	}
	return renderer.renderAssignment(state)
}

func renderOrder(state *renderState, order Order) error {
	renderer, ok := order.(orderRenderer)
	if !ok {
		return fmt.Errorf("dbx: unsupported order %T", order)
	}
	return renderer.renderOrder(state)
}

func (c Column[E, T]) renderOperand(state *renderState) (string, error) {
	meta := c.columnRef()
	var builder renderBuffer
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	builder.writeString(state.dialect.QuoteIdent(table))
	builder.writeByte('.')
	builder.writeString(state.dialect.QuoteIdent(meta.Name))
	return builder.String(), builder.Err("render column operand")
}

func (o valueOperand[T]) renderOperand(state *renderState) (string, error) {
	return state.bind(o.Value), nil
}

func (o columnOperand[T]) renderOperand(state *renderState) (string, error) {
	meta := o.Column.columnRef()
	var builder renderBuffer
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	builder.writeString(state.dialect.QuoteIdent(table))
	builder.writeByte('.')
	builder.writeString(state.dialect.QuoteIdent(meta.Name))
	return builder.String(), builder.Err("render column operand")
}

func (p comparisonPredicate) renderPredicate(state *renderState) error {
	left, err := p.Left.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(left)
	if p.Op == OpIs || p.Op == OpIsNot {
		state.writeByte(' ')
		state.writeString(string(p.Op))
		state.writeString(" NULL")
		return nil
	}
	operand, err := renderOperandValue(state, p.Right)
	if err != nil {
		return err
	}
	state.writeByte(' ')
	state.writeString(string(p.Op))
	state.writeByte(' ')
	state.writeString(operand)
	return nil
}

func (p logicalPredicate) renderPredicate(state *renderState) error {
	if len(p.Predicates) == 0 {
		return errors.New("dbx: logical predicate requires nested predicates")
	}
	state.writeByte('(')
	for i, predicate := range p.Predicates {
		if i > 0 {
			state.writeByte(' ')
			state.writeString(string(p.Op))
			state.writeByte(' ')
		}
		if err := renderPredicate(state, predicate); err != nil {
			return err
		}
	}
	state.writeByte(')')
	return nil
}

func (p notPredicate) renderPredicate(state *renderState) error {
	if p.Predicate == nil {
		return errors.New("dbx: NOT predicate requires nested predicate")
	}
	state.writeString("NOT (")
	if err := renderPredicate(state, p.Predicate); err != nil {
		return err
	}
	state.writeByte(')')
	return nil
}

func (p existsPredicate) renderPredicate(state *renderState) error {
	if p.Query == nil {
		return errors.New("dbx: EXISTS predicate requires subquery")
	}
	state.writeString("EXISTS (")
	if err := renderSelectStatement(state, p.Query); err != nil {
		return err
	}
	state.writeByte(')')
	return nil
}

func (a columnAssignment[E, T]) assignmentColumn() ColumnMeta {
	return a.Column.columnRef()
}

func (a columnAssignment[E, T]) renderAssignment(state *renderState) error {
	state.writeQuotedIdent(a.Column.Name())
	state.writeString(" = ")
	operand, err := renderOperandValue(state, a.Value)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (a columnAssignment[E, T]) renderAssignmentValue(state *renderState) error {
	operand, err := renderOperandValue(state, a.Value)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (o columnOrder[E, T]) renderOrder(state *renderState) error {
	state.renderColumn(o.Column.columnRef())
	if o.Descending {
		state.writeString(" DESC")
		return nil
	}
	state.writeString(" ASC")
	return nil
}

func (o expressionOrder) renderOrder(state *renderState) error {
	operand, err := o.Expr.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(operand)
	if o.Descending {
		state.writeString(" DESC")
		return nil
	}
	state.writeString(" ASC")
	return nil
}

func (a Aggregate[T]) renderOperand(state *renderState) (string, error) {
	var builder renderBuffer
	builder.writeString(string(a.Function))
	builder.writeByte('(')
	if a.Distinct {
		builder.writeString("DISTINCT ")
	}
	if a.star {
		builder.writeByte('*')
	} else {
		if a.Expr == nil {
			return "", fmt.Errorf("dbx: aggregate %s requires expression", a.Function)
		}
		operand, err := a.Expr.renderOperand(state)
		if err != nil {
			return "", err
		}
		builder.writeString(operand)
	}
	builder.writeByte(')')
	return builder.String(), builder.Err("render aggregate operand")
}

func (a Aggregate[T]) renderSelectItem(state *renderState) error {
	operand, err := a.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (c CaseExpression[T]) renderOperand(state *renderState) (string, error) {
	if len(c.Branches) == 0 {
		return "", errors.New("dbx: CASE expression requires at least one WHEN branch")
	}

	var builder renderBuffer
	builder.writeString("CASE")
	for _, branch := range c.Branches {
		if branch.Predicate == nil {
			return "", errors.New("dbx: CASE branch requires predicate")
		}
		builder.writeString(" WHEN ")
		predicateSQL, err := renderPredicateValue(state, branch.Predicate)
		if err != nil {
			return "", err
		}
		builder.writeString(predicateSQL)
		builder.writeString(" THEN ")
		valueSQL, err := renderOperandValue(state, branch.Value)
		if err != nil {
			return "", err
		}
		builder.writeString(valueSQL)
	}
	if c.Else != nil {
		builder.writeString(" ELSE ")
		elseSQL, err := renderOperandValue(state, c.Else)
		if err != nil {
			return "", err
		}
		builder.writeString(elseSQL)
	}
	builder.writeString(" END")
	return builder.String(), builder.Err("render case operand")
}

func (c CaseExpression[T]) renderSelectItem(state *renderState) error {
	operand, err := c.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (o excludedColumnOperand[T]) renderOperand(state *renderState) (string, error) {
	f := dialectFeatures(state.dialect)
	quoted := state.dialect.QuoteIdent(o.Column.Name)
	switch f.ExcludedRefStyle {
	case "excluded":
		return "EXCLUDED." + quoted, nil
	case "values":
		return "VALUES(" + quoted + ")", nil
	default:
		return "", fmt.Errorf("dbx: excluded assignment is not supported for dialect %s", state.dialect.Name())
	}
}

func (a aliasedSelectItem) renderSelectItem(state *renderState) error {
	if a.Item == nil {
		return errors.New("dbx: aliased select item requires value")
	}
	switch renderer := a.Item.(type) {
	case selectItemRenderer:
		if err := renderer.renderSelectItem(state); err != nil {
			return err
		}
	case operandRenderer:
		operand, err := renderer.renderOperand(state)
		if err != nil {
			return err
		}
		state.writeString(operand)
	default:
		return fmt.Errorf("dbx: unsupported aliased select item %T", a.Item)
	}
	if strings.TrimSpace(a.Alias) != "" {
		state.writeString(" AS ")
		state.writeQuotedIdent(strings.TrimSpace(a.Alias))
	}
	return nil
}

type subqueryOperand struct {
	Query *SelectQuery
}

func (subqueryOperand) expressionNode() {}

func (s subqueryOperand) renderOperand(state *renderState) (string, error) {
	if s.Query == nil {
		return "", errors.New("dbx: subquery is nil")
	}
	original := state.buf
	var builder strings.Builder
	state.buf = builder
	if err := renderSelectStatement(state, s.Query); err != nil {
		state.buf = original
		return "", err
	}
	rendered := state.buf.String()
	state.buf = original
	return "(" + rendered + ")", nil
}

func renderPredicateValue(state *renderState, predicate Predicate) (string, error) {
	original := state.buf
	var builder strings.Builder
	state.buf = builder
	if err := renderPredicate(state, predicate); err != nil {
		state.buf = original
		return "", err
	}
	rendered := state.buf.String()
	state.buf = original
	return rendered, nil
}

func renderOperandValue(state *renderState, value any) (string, error) {
	if renderer, ok := value.(operandRenderer); ok {
		return renderer.renderOperand(state)
	}
	if values, ok := value.([]any); ok {
		return renderAnySliceOperand(state, values)
	}
	return state.bind(value), nil
}

func renderAnySliceOperand(state *renderState, values []any) (string, error) {
	if len(values) == 0 {
		return "", errors.New("dbx: IN operand cannot be empty")
	}
	var builder renderBuffer
	builder.writeByte('(')
	for i, value := range values {
		if i > 0 {
			builder.writeString(", ")
		}
		builder.writeString(state.bind(value))
	}
	builder.writeByte(')')
	return builder.String(), builder.Err("render slice operand")
}

func normalizedInsertRows(q *InsertQuery) [][]Assignment {
	if len(q.Rows) > 0 {
		return q.Rows
	}
	if len(q.Assignments) > 0 {
		return [][]Assignment{q.Assignments}
	}
	return nil
}

func resolveInsertColumns(q *InsertQuery, rows [][]Assignment) ([]ColumnMeta, error) {
	if len(q.TargetColumns) > 0 {
		return resolveTargetColumns(q.TargetColumns)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	columns := collectionx.NewListWithCapacity[ColumnMeta](len(rows[0]))
	for _, assignment := range rows[0] {
		renderer, ok := assignment.(insertAssignmentRenderer)
		if !ok {
			return nil, fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
		}
		columns.Add(renderer.assignmentColumn())
	}
	return columns.Values(), nil
}

func resolveTargetColumns(expressions []Expression) ([]ColumnMeta, error) {
	columns := collectionx.NewListWithCapacity[ColumnMeta](len(expressions))
	for _, expression := range expressions {
		column, ok := expression.(columnAccessor)
		if !ok {
			return nil, fmt.Errorf("dbx: unsupported target column expression %T", expression)
		}
		columns.Add(column.columnRef())
	}
	return columns.Values(), nil
}

func orderInsertRows(columns []ColumnMeta, rows [][]Assignment) ([][]Assignment, error) {
	orderedRows := collectionx.NewListWithCapacity[[]Assignment](len(rows))
	for _, row := range rows {
		assignmentsByColumn := collectionx.NewMapWithCapacity[string, Assignment](len(row))
		for _, assignment := range row {
			renderer, ok := assignment.(insertAssignmentRenderer)
			if !ok {
				return nil, fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
			}
			assignmentsByColumn.Set(renderer.assignmentColumn().Name, assignment)
		}
		orderedRow := collectionx.NewListWithCapacity[Assignment](len(columns))
		for i := range columns {
			column := &columns[i]
			assignment, ok := assignmentsByColumn.Get(column.Name)
			if !ok {
				return nil, fmt.Errorf("dbx: missing value for insert column %s", column.Name)
			}
			orderedRow.Add(assignment)
		}
		orderedRows.Add(orderedRow.Values())
	}
	return orderedRows.Values(), nil
}

func renderUpsert(state *renderState, q *InsertQuery) error {
	if q.Upsert == nil {
		return nil
	}
	f := dialectFeatures(state.dialect)
	switch f.UpsertVariant {
	case "on_conflict":
		return renderUpsertOnConflict(state, q)
	case "on_duplicate_key":
		return renderUpsertOnDuplicateKey(state, q)
	default:
		return fmt.Errorf("dbx: upsert is not supported for dialect %s", state.dialect.Name())
	}
}

func renderUpsertOnConflict(state *renderState, q *InsertQuery) error {
	state.writeString(" ON CONFLICT")
	if len(q.Upsert.Targets) > 0 {
		state.writeString(" (")
		for i, target := range q.Upsert.Targets {
			if i > 0 {
				state.writeString(", ")
			}
			if column, ok := target.(columnAccessor); ok {
				state.writeQuotedIdent(column.columnRef().Name)
				continue
			}
			operand, err := renderOperandValue(state, target)
			if err != nil {
				return err
			}
			state.writeString(operand)
		}
		state.writeByte(')')
	}
	if q.Upsert.DoNothing {
		state.writeString(" DO NOTHING")
		return nil
	}
	if len(q.Upsert.Assignments) == 0 {
		return errors.New("dbx: upsert update requires assignments")
	}
	if len(q.Upsert.Targets) == 0 {
		return errors.New("dbx: upsert update requires conflict targets")
	}
	state.writeString(" DO UPDATE SET ")
	return renderUpsertAssignments(state, q.Upsert.Assignments)
}

func renderUpsertOnDuplicateKey(state *renderState, q *InsertQuery) error {
	if q.Upsert.DoNothing {
		return nil
	}
	if len(q.Upsert.Assignments) == 0 {
		return errors.New("dbx: upsert update requires assignments")
	}
	state.writeString(" ON DUPLICATE KEY UPDATE ")
	return renderUpsertAssignments(state, q.Upsert.Assignments)
}

func renderUpsertAssignments(state *renderState, assignments []Assignment) error {
	for i, assignment := range assignments {
		if i > 0 {
			state.writeString(", ")
		}
		if err := renderAssignment(state, assignment); err != nil {
			return err
		}
	}
	return nil
}

func dialectFeatures(d dialect.Dialect) dialect.QueryFeatures {
	if p, ok := d.(dialect.QueryFeaturesProvider); ok {
		return p.QueryFeatures()
	}
	return dialect.DefaultQueryFeatures(d.Name())
}

func renderReturning(state *renderState, items []SelectItem) error {
	if len(items) == 0 {
		return nil
	}
	if !dialectFeatures(state.dialect).SupportsReturning {
		return fmt.Errorf("dbx: RETURNING is not supported for dialect %s", state.dialect.Name())
	}
	state.writeString(" RETURNING ")
	for i, item := range items {
		if i > 0 {
			state.writeString(", ")
		}
		if err := renderSelectItem(state, item); err != nil {
			return err
		}
	}
	return nil
}
