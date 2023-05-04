package resource

import rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"

type Conditions []rtv1.Condition

func (cs *Conditions) UpsertCondition(cond rtv1.Condition) {
	for idx, el := range *cs {
		if el.Type == cond.Type {
			(*cs)[idx] = cond
			return
		}
	}
	*cs = append(*cs, cond)
}

func (cs *Conditions) UpsertConditionMessage(cond rtv1.Condition) {
	for idx, el := range *cs {
		if el.Type == cond.Type {
			(*cs)[idx].Message += ", " + cond.Message
			return
		}
	}
	*cs = append(*cs, cond)
}

func (cs *Conditions) JoinConditions(conds *Conditions) {
	for _, el := range *conds {
		cs.UpsertCondition(el)
	}
}

func (cs *Conditions) RemoveCondition(ct rtv1.ConditionType) {
	for idx, el := range *cs {
		if el.Type == ct {
			*cs = append((*cs)[:idx], (*cs)[idx+1:]...)
			return
		}
	}
}
