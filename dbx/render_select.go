package dbx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

func (q *SelectQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: select query is nil")
	}
	if q.FromItem.Name() == "" {
		return BoundQuery{}, errors.New("dbx: select query requires FROM")
	}
	if q.Items.Len() == 0 {
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
	if q.Unions.Len() == 0 {
		return renderSelectQuery(state, q)
	}

	if err := renderSelectQueryWithoutTail(state, q); err != nil {
		return err
	}
	var renderErr error
	q.Unions.Range(func(_ int, union UnionClause) bool {
		if union.Query == nil {
			renderErr = errors.New("dbx: union query is nil")
			return false
		}
		if union.All {
			state.writeString(" UNION ALL ")
		} else {
			state.writeString(" UNION ")
		}
		if err := renderUnionQuery(state, union.Query); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	if renderErr != nil {
		return renderErr
	}
	return renderSelectTail(state, q)
}

func renderCTEs(state *renderState, ctes collectionx.List[CTE]) error {
	if ctes.Len() == 0 {
		return nil
	}
	state.writeString("WITH ")
	var renderErr error
	ctes.Range(func(index int, cte CTE) bool {
		if strings.TrimSpace(cte.Name) == "" {
			renderErr = errors.New("dbx: cte name cannot be empty")
			return false
		}
		if cte.Query == nil {
			renderErr = fmt.Errorf("dbx: cte %s requires query", cte.Name)
			return false
		}
		if index > 0 {
			state.writeString(", ")
		}
		state.writeQuotedIdent(strings.TrimSpace(cte.Name))
		state.writeString(" AS (")
		if err := renderSelectStatement(state, cte.Query); err != nil {
			renderErr = err
			return false
		}
		state.writeByte(')')
		return true
	})
	if renderErr != nil {
		return renderErr
	}
	state.writeByte(' ')
	return nil
}

func renderUnionQuery(state *renderState, q *SelectQuery) error {
	if q.CTEs.Len() > 0 || q.Unions.Len() > 0 || q.Orders.Len() > 0 || q.LimitN != nil || q.OffsetN != nil {
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
	return renderSelectHaving(state, q)
}

func renderSelectDistinct(state *renderState, q *SelectQuery) error {
	state.writeString("SELECT ")
	if q.Distinct {
		state.writeString("DISTINCT ")
	}
	return nil
}

func renderSelectItems(state *renderState, q *SelectQuery) error {
	var renderErr error
	q.Items.Range(func(index int, item SelectItem) bool {
		if index > 0 {
			state.writeString(", ")
		}
		if err := renderSelectItem(state, item); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	return renderErr
}

func renderSelectFrom(state *renderState, q *SelectQuery) error {
	state.writeString(" FROM ")
	state.renderTable(q.FromItem)
	return nil
}

func renderSelectJoins(state *renderState, q *SelectQuery) error {
	var renderErr error
	q.Joins.Range(func(_ int, join Join) bool {
		state.writeByte(' ')
		state.writeString(string(join.Type))
		state.writeString(" JOIN ")
		state.renderTable(join.Table)
		if join.Predicate == nil {
			return true
		}
		state.writeString(" ON ")
		if err := renderPredicate(state, join.Predicate); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	return renderErr
}

func renderSelectWhere(state *renderState, q *SelectQuery) error {
	if q.WhereExp == nil {
		return nil
	}
	state.writeString(" WHERE ")
	return renderPredicate(state, q.WhereExp)
}

func renderSelectGroupBy(state *renderState, q *SelectQuery) error {
	if q.Groups.Len() == 0 {
		return nil
	}
	state.writeString(" GROUP BY ")
	var renderErr error
	q.Groups.Range(func(index int, group Expression) bool {
		if index > 0 {
			state.writeString(", ")
		}
		operand, err := renderOperandValue(state, group)
		if err != nil {
			renderErr = err
			return false
		}
		state.writeString(operand)
		return true
	})
	return renderErr
}

func renderSelectHaving(state *renderState, q *SelectQuery) error {
	if q.HavingExp == nil {
		return nil
	}
	state.writeString(" HAVING ")
	return renderPredicate(state, q.HavingExp)
}

func renderSelectTail(state *renderState, q *SelectQuery) error {
	if err := renderSelectOrders(state, q.Orders); err != nil {
		return err
	}
	return renderSelectLimitOffset(state, q)
}

func renderSelectOrders(state *renderState, orders collectionx.List[Order]) error {
	if orders.Len() == 0 {
		return nil
	}
	state.writeString(" ORDER BY ")
	var renderErr error
	orders.Range(func(index int, order Order) bool {
		if index > 0 {
			state.writeString(", ")
		}
		if err := renderOrder(state, order); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	return renderErr
}

func renderSelectLimitOffset(state *renderState, q *SelectQuery) error {
	clause, err := state.dialect.RenderLimitOffset(q.LimitN, q.OffsetN)
	if err != nil {
		return fmt.Errorf("dbx: render limit offset: %w", err)
	}
	if clause == "" {
		return nil
	}
	state.writeByte(' ')
	state.writeString(clause)
	return nil
}
