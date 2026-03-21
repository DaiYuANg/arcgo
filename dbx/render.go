package dbx

import (
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
	dialect dialect.Dialect
	buf     strings.Builder
	args    []any
}

func (s *renderState) bind(value any) string {
	s.args = append(s.args, value)
	return s.dialect.BindVar(len(s.args))
}

func (s *renderState) writeQuotedIdent(name string) {
	s.buf.WriteString(s.dialect.QuoteIdent(name))
}

func (s *renderState) writeQualifiedIdent(table, column string) {
	if table != "" {
		s.writeQuotedIdent(table)
		s.buf.WriteByte('.')
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
		s.buf.WriteString(" AS ")
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
		return BoundQuery{}, fmt.Errorf("dbx: select query is nil")
	}
	if q.FromItem.Name() == "" {
		return BoundQuery{}, fmt.Errorf("dbx: select query requires FROM")
	}
	if len(q.Items) == 0 {
		return BoundQuery{}, fmt.Errorf("dbx: select query requires at least one item")
	}

	state := &renderState{dialect: d, args: make([]any, 0, 8)}
	if err := renderSelectStatement(state, q); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
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
			return fmt.Errorf("dbx: union query is nil")
		}
		if union.All {
			state.buf.WriteString(" UNION ALL ")
		} else {
			state.buf.WriteString(" UNION ")
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
	state.buf.WriteString("WITH ")
	for index, cte := range ctes {
		if strings.TrimSpace(cte.Name) == "" {
			return fmt.Errorf("dbx: cte name cannot be empty")
		}
		if cte.Query == nil {
			return fmt.Errorf("dbx: cte %s requires query", cte.Name)
		}
		if index > 0 {
			state.buf.WriteString(", ")
		}
		state.writeQuotedIdent(strings.TrimSpace(cte.Name))
		state.buf.WriteString(" AS (")
		if err := renderSelectStatement(state, cte.Query); err != nil {
			return err
		}
		state.buf.WriteByte(')')
	}
	state.buf.WriteByte(' ')
	return nil
}

func renderUnionQuery(state *renderState, q *SelectQuery) error {
	if len(q.CTEs) > 0 || len(q.Unions) > 0 || len(q.Orders) > 0 || q.LimitN != nil || q.OffsetN != nil {
		state.buf.WriteByte('(')
		if err := renderSelectStatement(state, q); err != nil {
			return err
		}
		state.buf.WriteByte(')')
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
	state.buf.WriteString("SELECT ")
	if q.Distinct {
		state.buf.WriteString("DISTINCT ")
	}
	for i, item := range q.Items {
		if i > 0 {
			state.buf.WriteString(", ")
		}
		if err := renderSelectItem(state, item); err != nil {
			return err
		}
	}

	state.buf.WriteString(" FROM ")
	state.renderTable(q.FromItem)
	for _, join := range q.Joins {
		state.buf.WriteByte(' ')
		state.buf.WriteString(string(join.Type))
		state.buf.WriteString(" JOIN ")
		state.renderTable(join.Table)
		if join.Predicate != nil {
			state.buf.WriteString(" ON ")
			if err := renderPredicate(state, join.Predicate); err != nil {
				return err
			}
		}
	}
	if q.WhereExp != nil {
		state.buf.WriteString(" WHERE ")
		if err := renderPredicate(state, q.WhereExp); err != nil {
			return err
		}
	}
	if len(q.Groups) > 0 {
		state.buf.WriteString(" GROUP BY ")
		for i, group := range q.Groups {
			if i > 0 {
				state.buf.WriteString(", ")
			}
			operand, err := renderOperandValue(state, group)
			if err != nil {
				return err
			}
			state.buf.WriteString(operand)
		}
	}
	if q.HavingExp != nil {
		state.buf.WriteString(" HAVING ")
		if err := renderPredicate(state, q.HavingExp); err != nil {
			return err
		}
	}
	return nil
}

func renderSelectTail(state *renderState, q *SelectQuery) error {
	if len(q.Orders) > 0 {
		state.buf.WriteString(" ORDER BY ")
		for i, order := range q.Orders {
			if i > 0 {
				state.buf.WriteString(", ")
			}
			if err := renderOrder(state, order); err != nil {
				return err
			}
		}
	}
	clause, err := state.dialect.RenderLimitOffset(q.LimitN, q.OffsetN)
	if err != nil {
		return err
	}
	if clause != "" {
		state.buf.WriteByte(' ')
		state.buf.WriteString(clause)
	}
	return nil
}

func (q *InsertQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, fmt.Errorf("dbx: insert query is nil")
	}
	if q.Into.Name() == "" {
		return BoundQuery{}, fmt.Errorf("dbx: insert query requires target table")
	}
	rows := normalizedInsertRows(q)
	if len(rows) == 0 && q.Source == nil {
		return BoundQuery{}, fmt.Errorf("dbx: insert query requires values or source query")
	}
	if len(rows) > 0 && q.Source != nil {
		return BoundQuery{}, fmt.Errorf("dbx: insert query cannot combine values and source query")
	}
	if q.Source != nil && len(q.TargetColumns) == 0 {
		return BoundQuery{}, fmt.Errorf("dbx: insert-select requires target columns")
	}

	state := &renderState{dialect: d, args: make([]any, 0, len(rows)*4)}
	features := dialectFeatures(d)
	if features.InsertIgnoreForUpsertNothing && q.Upsert != nil && q.Upsert.DoNothing {
		state.buf.WriteString("INSERT IGNORE INTO ")
	} else {
		state.buf.WriteString("INSERT INTO ")
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
	return state.BoundQuery(), nil
}

func renderInsertBody(state *renderState, q *InsertQuery, rows [][]Assignment) error {
	state.renderTable(q.Into)
	columns, err := resolveInsertColumns(q, rows)
	if err != nil {
		return err
	}
	if len(columns) > 0 {
		state.buf.WriteString(" (")
		for i, column := range columns {
			if i > 0 {
				state.buf.WriteString(", ")
			}
			state.writeQuotedIdent(column.Name)
		}
		state.buf.WriteByte(')')
	}
	if q.Source != nil {
		state.buf.WriteByte(' ')
		return renderSelectQuery(state, q.Source)
	}
	orderedRows, err := orderInsertRows(columns, rows)
	if err != nil {
		return err
	}
	state.buf.WriteString(" VALUES ")
	for rowIndex, row := range orderedRows {
		if rowIndex > 0 {
			state.buf.WriteString(", ")
		}
		state.buf.WriteByte('(')
		for colIndex, assignment := range row {
			renderer, ok := assignment.(insertAssignmentRenderer)
			if !ok {
				return fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
			}
			if colIndex > 0 {
				state.buf.WriteString(", ")
			}
			if err := renderer.renderAssignmentValue(state); err != nil {
				return err
			}
		}
		state.buf.WriteByte(')')
	}
	return nil
}

func (q *UpdateQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, fmt.Errorf("dbx: update query is nil")
	}
	if q.Table.Name() == "" {
		return BoundQuery{}, fmt.Errorf("dbx: update query requires target table")
	}
	if len(q.Assignments) == 0 {
		return BoundQuery{}, fmt.Errorf("dbx: update query requires assignments")
	}

	state := &renderState{dialect: d, args: make([]any, 0, len(q.Assignments))}
	state.buf.WriteString("UPDATE ")
	state.renderTable(q.Table)
	state.buf.WriteString(" SET ")
	for i, assignment := range q.Assignments {
		if i > 0 {
			state.buf.WriteString(", ")
		}
		if err := renderAssignment(state, assignment); err != nil {
			return BoundQuery{}, err
		}
	}
	if q.WhereExp != nil {
		state.buf.WriteString(" WHERE ")
		if err := renderPredicate(state, q.WhereExp); err != nil {
			return BoundQuery{}, err
		}
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func (q *DeleteQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, fmt.Errorf("dbx: delete query is nil")
	}
	if q.From.Name() == "" {
		return BoundQuery{}, fmt.Errorf("dbx: delete query requires target table")
	}

	state := &renderState{dialect: d, args: make([]any, 0, 4)}
	state.buf.WriteString("DELETE FROM ")
	state.renderTable(q.From)
	if q.WhereExp != nil {
		state.buf.WriteString(" WHERE ")
		if err := renderPredicate(state, q.WhereExp); err != nil {
			return BoundQuery{}, err
		}
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
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
	var builder strings.Builder
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	builder.WriteString(state.dialect.QuoteIdent(table))
	builder.WriteByte('.')
	builder.WriteString(state.dialect.QuoteIdent(meta.Name))
	return builder.String(), nil
}

func (o valueOperand[T]) renderOperand(state *renderState) (string, error) {
	return state.bind(o.Value), nil
}

func (o columnOperand[T]) renderOperand(state *renderState) (string, error) {
	meta := o.Column.columnRef()
	var builder strings.Builder
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	builder.WriteString(state.dialect.QuoteIdent(table))
	builder.WriteByte('.')
	builder.WriteString(state.dialect.QuoteIdent(meta.Name))
	return builder.String(), nil
}

func (p comparisonPredicate) renderPredicate(state *renderState) error {
	left, err := p.Left.renderOperand(state)
	if err != nil {
		return err
	}
	state.buf.WriteString(left)
	if p.Op == OpIs || p.Op == OpIsNot {
		state.buf.WriteByte(' ')
		state.buf.WriteString(string(p.Op))
		state.buf.WriteString(" NULL")
		return nil
	}
	operand, err := renderOperandValue(state, p.Right)
	if err != nil {
		return err
	}
	state.buf.WriteByte(' ')
	state.buf.WriteString(string(p.Op))
	state.buf.WriteByte(' ')
	state.buf.WriteString(operand)
	return nil
}

func (p logicalPredicate) renderPredicate(state *renderState) error {
	if len(p.Predicates) == 0 {
		return fmt.Errorf("dbx: logical predicate requires nested predicates")
	}
	state.buf.WriteByte('(')
	for i, predicate := range p.Predicates {
		if i > 0 {
			state.buf.WriteByte(' ')
			state.buf.WriteString(string(p.Op))
			state.buf.WriteByte(' ')
		}
		if err := renderPredicate(state, predicate); err != nil {
			return err
		}
	}
	state.buf.WriteByte(')')
	return nil
}

func (p notPredicate) renderPredicate(state *renderState) error {
	if p.Predicate == nil {
		return fmt.Errorf("dbx: NOT predicate requires nested predicate")
	}
	state.buf.WriteString("NOT (")
	if err := renderPredicate(state, p.Predicate); err != nil {
		return err
	}
	state.buf.WriteByte(')')
	return nil
}

func (p existsPredicate) renderPredicate(state *renderState) error {
	if p.Query == nil {
		return fmt.Errorf("dbx: EXISTS predicate requires subquery")
	}
	state.buf.WriteString("EXISTS (")
	if err := renderSelectStatement(state, p.Query); err != nil {
		return err
	}
	state.buf.WriteByte(')')
	return nil
}

func (a columnAssignment[E, T]) assignmentColumn() ColumnMeta {
	return a.Column.columnRef()
}

func (a columnAssignment[E, T]) renderAssignment(state *renderState) error {
	state.writeQuotedIdent(a.Column.Name())
	state.buf.WriteString(" = ")
	operand, err := renderOperandValue(state, a.Value)
	if err != nil {
		return err
	}
	state.buf.WriteString(operand)
	return nil
}

func (a columnAssignment[E, T]) renderAssignmentValue(state *renderState) error {
	operand, err := renderOperandValue(state, a.Value)
	if err != nil {
		return err
	}
	state.buf.WriteString(operand)
	return nil
}

func (o columnOrder[E, T]) renderOrder(state *renderState) error {
	state.renderColumn(o.Column.columnRef())
	if o.Descending {
		state.buf.WriteString(" DESC")
		return nil
	}
	state.buf.WriteString(" ASC")
	return nil
}

func (o expressionOrder) renderOrder(state *renderState) error {
	operand, err := o.Expr.renderOperand(state)
	if err != nil {
		return err
	}
	state.buf.WriteString(operand)
	if o.Descending {
		state.buf.WriteString(" DESC")
		return nil
	}
	state.buf.WriteString(" ASC")
	return nil
}

func (a Aggregate[T]) renderOperand(state *renderState) (string, error) {
	var builder strings.Builder
	builder.WriteString(string(a.Function))
	builder.WriteByte('(')
	if a.Distinct {
		builder.WriteString("DISTINCT ")
	}
	if a.star {
		builder.WriteByte('*')
	} else {
		if a.Expr == nil {
			return "", fmt.Errorf("dbx: aggregate %s requires expression", a.Function)
		}
		operand, err := a.Expr.renderOperand(state)
		if err != nil {
			return "", err
		}
		builder.WriteString(operand)
	}
	builder.WriteByte(')')
	return builder.String(), nil
}

func (a Aggregate[T]) renderSelectItem(state *renderState) error {
	operand, err := a.renderOperand(state)
	if err != nil {
		return err
	}
	state.buf.WriteString(operand)
	return nil
}

func (c CaseExpression[T]) renderOperand(state *renderState) (string, error) {
	if len(c.Branches) == 0 {
		return "", fmt.Errorf("dbx: CASE expression requires at least one WHEN branch")
	}

	var builder strings.Builder
	builder.WriteString("CASE")
	for _, branch := range c.Branches {
		if branch.Predicate == nil {
			return "", fmt.Errorf("dbx: CASE branch requires predicate")
		}
		builder.WriteString(" WHEN ")
		predicateSQL, err := renderPredicateValue(state, branch.Predicate)
		if err != nil {
			return "", err
		}
		builder.WriteString(predicateSQL)
		builder.WriteString(" THEN ")
		valueSQL, err := renderOperandValue(state, branch.Value)
		if err != nil {
			return "", err
		}
		builder.WriteString(valueSQL)
	}
	if c.Else != nil {
		builder.WriteString(" ELSE ")
		elseSQL, err := renderOperandValue(state, c.Else)
		if err != nil {
			return "", err
		}
		builder.WriteString(elseSQL)
	}
	builder.WriteString(" END")
	return builder.String(), nil
}

func (c CaseExpression[T]) renderSelectItem(state *renderState) error {
	operand, err := c.renderOperand(state)
	if err != nil {
		return err
	}
	state.buf.WriteString(operand)
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
		return fmt.Errorf("dbx: aliased select item requires value")
	}
	if renderer, ok := a.Item.(selectItemRenderer); ok {
		if err := renderer.renderSelectItem(state); err != nil {
			return err
		}
	} else if renderer, ok := a.Item.(operandRenderer); ok {
		operand, err := renderer.renderOperand(state)
		if err != nil {
			return err
		}
		state.buf.WriteString(operand)
	} else {
		return fmt.Errorf("dbx: unsupported aliased select item %T", a.Item)
	}
	if strings.TrimSpace(a.Alias) != "" {
		state.buf.WriteString(" AS ")
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
		return "", fmt.Errorf("dbx: subquery is nil")
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
		return "", fmt.Errorf("dbx: IN operand cannot be empty")
	}
	var builder strings.Builder
	builder.WriteByte('(')
	for i, value := range values {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(state.bind(value))
	}
	builder.WriteByte(')')
	return builder.String(), nil
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
		for _, column := range columns {
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
		state.buf.WriteString(" ON CONFLICT")
		if len(q.Upsert.Targets) > 0 {
			state.buf.WriteString(" (")
			for i, target := range q.Upsert.Targets {
				if i > 0 {
					state.buf.WriteString(", ")
				}
				if column, ok := target.(columnAccessor); ok {
					state.writeQuotedIdent(column.columnRef().Name)
					continue
				}
				operand, err := renderOperandValue(state, target)
				if err != nil {
					return err
				}
				state.buf.WriteString(operand)
			}
			state.buf.WriteByte(')')
		}
		if q.Upsert.DoNothing {
			state.buf.WriteString(" DO NOTHING")
			return nil
		}
		if len(q.Upsert.Assignments) == 0 {
			return fmt.Errorf("dbx: upsert update requires assignments")
		}
		if len(q.Upsert.Targets) == 0 {
			return fmt.Errorf("dbx: upsert update requires conflict targets")
		}
		state.buf.WriteString(" DO UPDATE SET ")
		for i, assignment := range q.Upsert.Assignments {
			if i > 0 {
				state.buf.WriteString(", ")
			}
			if err := renderAssignment(state, assignment); err != nil {
				return err
			}
		}
		return nil
	case "on_duplicate_key":
		if q.Upsert.DoNothing {
			return nil
		}
		if len(q.Upsert.Assignments) == 0 {
			return fmt.Errorf("dbx: upsert update requires assignments")
		}
		state.buf.WriteString(" ON DUPLICATE KEY UPDATE ")
		for i, assignment := range q.Upsert.Assignments {
			if i > 0 {
				state.buf.WriteString(", ")
			}
			if err := renderAssignment(state, assignment); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("dbx: upsert is not supported for dialect %s", state.dialect.Name())
	}
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
	state.buf.WriteString(" RETURNING ")
	for i, item := range items {
		if i > 0 {
			state.buf.WriteString(", ")
		}
		if err := renderSelectItem(state, item); err != nil {
			return err
		}
	}
	return nil
}
