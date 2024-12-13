package tx

import (
	"fmt"
	"math/big"

	"github.com/cometbft/cometbft/libs/pubsub/query/syntax"
	cometbftindexer "github.com/cometbft/cometbft/state/indexer"
	cmttypes "github.com/cometbft/cometbft/types"
)

type HeightInfo struct {
	HeightRange     cometbftindexer.QueryRange
	Height          int64
	heightEqIdx     int
	onlyHeightRange bool
	onlyHeightEq    bool
}

func DedupHeight(conditions []syntax.Condition) (dedupConditions []syntax.Condition, heightInfo HeightInfo) {
	heightInfo.heightEqIdx = -1
	heightRangeExists := false
	found := false
	var heightCondition []syntax.Condition
	heightInfo.onlyHeightEq = true
	heightInfo.onlyHeightRange = true
	for _, c := range conditions {
		if c.Tag == cmttypes.TxHeightKey {
			if c.Op == syntax.TEq {
				if heightRangeExists || found {
					continue
				}
				hFloat := c.Arg.Number()
				if hFloat != nil {
					h, _ := hFloat.Int64()
					heightInfo.Height = h
					found = true
					heightCondition = append(heightCondition, c)
				}
			} else {
				heightInfo.onlyHeightEq = false
				heightRangeExists = true
				dedupConditions = append(dedupConditions, c)
			}
		} else {
			heightInfo.onlyHeightRange = false
			heightInfo.onlyHeightEq = false
			dedupConditions = append(dedupConditions, c)
		}
	}
	if !heightRangeExists && len(heightCondition) != 0 {
		heightInfo.heightEqIdx = len(dedupConditions)
		heightInfo.onlyHeightRange = false
		dedupConditions = append(dedupConditions, heightCondition...)
	} else {
		// If we found a range make sure we set the height idx to -1 as the height equality
		// will be removed
		heightInfo.heightEqIdx = -1
		heightInfo.Height = 0
		heightInfo.onlyHeightEq = false
	}
	return dedupConditions, heightInfo
}

func CreateHeightRangeWhereConditions(heightInfo HeightInfo) (whereConditions string, vals []interface{}, argsCount *int) {
	// args count is used to increment parameterized arguments
	initialCount := 1
	argsCount = &initialCount
	// prioritize range conditions
	if isHeightRangeNotEmpty(heightInfo.HeightRange) {
		value := heightInfo.HeightRange
		ops, values := detectQueryRangeBound(value)
		whereConditions += "WHERE"
		for i, operator := range ops {
			if i == len(ops)-1 {
				whereConditions += fmt.Sprintf(" height %s $%d", operator, *argsCount)
			} else {
				whereConditions += fmt.Sprintf(" height %s $%d AND", operator, *argsCount)
			}

			*argsCount++
		}
		vals = values
		return whereConditions, vals, argsCount
	}
	// if there's no range, and has eq condition -> handle it
	if heightInfo.Height != 0 {
		return fmt.Sprintf("WHERE height = $%d", *argsCount), []interface{}{heightInfo.Height}, argsCount
	}
	return "", nil, &initialCount
}

func isHeightRangeNotEmpty(heightRange cometbftindexer.QueryRange) bool {
	return heightRange.LowerBound != nil || heightRange.UpperBound != nil
}

func CreateNonHeightConditionFilterTable(conditions []syntax.Condition, argsCount *int) (filterTableClause string, vals []interface{}, err error) {
	filterTableClause += "filtered_tx_event_attributes as ("
	filterTxs := func() string {
		return fmt.Sprintln(`select distinct tx_id 
		FROM events 
		JOIN filtered_heights fh ON (fh.rowid = events.tx_id) 
		JOIN attributes ON (events.rowid = attributes.event_id)`)
	}
	hasNonheightCondition := false
	for i, condition := range conditions {
		// ignore since we already covered tx.height elsewhere
		if condition.Tag == cmttypes.TxHeightKey {
			continue
		}
		hasNonheightCondition = true
		whereClause := fmt.Sprintf("%sWHERE composite_key = $%d \n", filterTxs(), *argsCount)
		*argsCount++
		vals = append(vals, condition.Tag)
		whereValueClause, val, err := matchNonHeightCondition(condition, argsCount)
		if err != nil {
			return "", vals, err
		}
		whereClause += whereValueClause
		vals = append(vals, val)
		filterTableClause += whereClause

		// if it's not the last condition -> add INTERSECT keyword to intersect the tables for AND condition
		// TODO: If we allow OR keyword -> switch case to UNION
		if i < len(conditions)-1 {
			filterTableClause += "INTERSECT\n"
		}
	}
	// empty table clause, meaning that there are no other clauses -> return filtered_tx_ids bare minimum
	if !hasNonheightCondition {
		return fmt.Sprintf("%s\n%s)", filterTableClause, filterTxs()), vals, nil
	}
	return fmt.Sprintf("%s)\n", filterTableClause), vals, nil
}

func matchNonHeightCondition(condition syntax.Condition, argsCount *int) (whereClause string, val interface{}, err error) {
	opStr, err := convertOpToOpStr(condition.Op)
	if err != nil {
		return "", nil, err
	}
	val = conditionArg(condition)
	clause := fmt.Sprintf("AND value %s $%d \n", opStr, *argsCount)
	// for numbers, we cast value as numeric
	if condition.Arg.Type == syntax.TNumber {
		clause = fmt.Sprintf("AND value::numeric %s $%d \n", opStr, *argsCount)
	}
	*argsCount++
	return clause, val, nil
}

func detectQueryRangeBound(value cometbftindexer.QueryRange) (ops []string, vals []interface{}) {
	if value.LowerBound != nil {
		operator := ">"
		if value.IncludeLowerBound {
			operator = ">="
		}
		ops = append(ops, operator)
		val, _ := value.LowerBound.(*big.Float).Int64()
		vals = append(vals, val)
	}
	if value.UpperBound != nil {
		operator := "<"
		if value.IncludeUpperBound {
			operator = "<="
		}
		ops = append(ops, operator)
		upper, _ := value.UpperBound.(*big.Float).Int64()
		vals = append(vals, upper)
	}
	return ops, vals
}

func convertOpToOpStr(op syntax.Token) (string, error) {
	switch op {
	case syntax.TEq:
		return "=", nil
	case syntax.TGeq:
		return ">=", nil
	case syntax.TLeq:
		return "<=", nil
	case syntax.TLt:
		return "<", nil
	case syntax.TGt:
		return ">", nil
	default:
		return "", fmt.Errorf("error converting op to op str. The op doesn't match any defined op")
	}
}

func conditionArg(c syntax.Condition) interface{} {
	if c.Arg == nil {
		return nil
	}
	switch c.Arg.Type {
	case syntax.TNumber:
		num, _ := c.Arg.Number().Int64()
		return num
	case syntax.TTime, syntax.TDate:
		return c.Arg.Time()
	default:
		return c.Arg.Value() // string
	}
}
