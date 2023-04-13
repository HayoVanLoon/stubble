package stuble

import (
	"context"
)

type inMemory struct {
	rules []Rule
}

func (p inMemory) ListRules(context.Context) ([]Rule, error) {
	rs := make([]Rule, len(p.rules), len(p.rules))
	copy(rs, p.rules)
	return rs, nil
}

func (p inMemory) SaveRule(ctx context.Context, ru Rule) error {
	for i := range p.rules {
		if ru.EqualMatch(p.rules[i]) {
			p.rules[i] = ru
			return nil
		}
	}
	p.rules = append(p.rules, ru)
	return nil
}
