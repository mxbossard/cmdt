package parser

import (
	"fmt"
	"strings"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/errorz"
)

/**
## Rules
OPEN: comment gérer ces alias: @help et -h ?
OPEN: comment gérer help qui est particulier car tout est ME en mode doc derriere help ?
OPEN: comment gérer le multi value ?

## Parser rules
- do not allow rules after @--
- consider all args not "rule parsable" as command and args to test
- while no command is parsed allow prefix to be - for 1 char aliases or -- for aliases ?
*/

type configurer interface {
	Mutate(cfg *model.Config, assertions *[]model.Assertion)
}

type ruleMatcher interface {
	Name() string
	Match(prefix string, args []string) (int, configurer, errorz.Aggregated)
	//MutateIfMatch(args []string, cfg *model.Config, assertions *[]model.Assertion) errorz.Aggregated
}

type operator[T any] struct {
	op         string
	mapper     *mapper[T]
	validaters []*validater[T]
}

type rule[T any] struct {
	name        string
	operators   []*operator[T]
	mutater     *configMutater[T]
	aliases     []string
	prefixMask  string // @: Standard behavior: rules prefixed by PREFIX ; -: allow - and -- ;
	assertion   bool
	multiValued bool
}

func (r rule[T]) Name() string {
	return r.name
}

func (r rule[T]) Match(prefix string, args []string) (n int, cfg configurer, agg errorz.Aggregated) {
	// Verify if supplied args match the rule
	// If so, return the args count matched, the errors encountered and an object abale to mutate the config
	// FIXME: should return an object able to mutate the config

	if len(args) == 0 {
		return
	}

	// 1- identify if prefix match and which prefix is it
	var matchPrefix bool
	if strings.Contains(r.prefixMask, "@") {
		// arg can begin with prefix
		if strings.HasPrefix(args[0], prefix) {
			matchPrefix = true
		}
	}
	if strings.Contains(r.prefixMask, "-") {
		// arg can begin with - or --
		if strings.HasPrefix(args[0], "--") {
			prefix = "--"
			matchPrefix = true
		} else if strings.HasPrefix(args[0], "-") {
			prefix = "-"
			matchPrefix = true
		}
	}

	if !matchPrefix {
		return 0, nil, agg
	}

	// 2- identifiy if rule name and operator match
	if r.operators == nil {
		// To match must not have an operator nor a value
		if args[0] == prefix+r.name {
			n = 1
			cfg2 := &config2[T]{}
			cfg2.op = ""
			cfg2.prefix = prefix
			cfg2.rule = &r
			cfg2.value = ""
			cfg = cfg2
			return
		}
	}

	var matchingOpLen int
	var matchingOp *operator[T]
	var value string
	for _, op := range r.operators {
		// Multiple op could match, must keep longest op
		if op.op == " " {
			// Special case: space operator
			if args[0] == prefix+r.name {
				if len(args) == 1 {
					n = 1
					agg.Add(fmt.Errorf("missing arg after space operator"))
					return
				}
				matchingOp = op
				n = 2
				value = args[1]
			}
			break
		}
		if len(op.op) > matchingOpLen && strings.HasPrefix(args[0], prefix+r.name+op.op) {
			matchingOpLen = len(op.op)
			matchingOp = op
			n = 1
			value = strings.TrimPrefix(args[0], prefix+r.name+op.op)
			return
		}
	}

	if matchingOp == nil {
		// FIXME: if one alias match but no operator should hint with an error
		return 0, nil, agg
	}

	// 3- validate value
	mapper := *matchingOp.mapper
	mappedValue, err := mapper(matchingOp.op, value)
	if err != nil {
		agg.Add(err)
	} else {
		for _, validaterPtr := range matchingOp.validaters {
			validater := *validaterPtr
			err := validater(mappedValue)
			if err != nil {
				agg.Add(err)
			}
		}
	}

	if agg.GotError() {
		return n, nil, agg
	}
	cfg2 := &config2[T]{}
	cfg2.op = matchingOp.op
	cfg2.prefix = prefix
	cfg2.rule = &r
	cfg2.value = value
	cfg2.mappedValue = mappedValue
	cfg = cfg2

	return
}

func (r rule[T]) MutateIfMatch(args []string, cfg *model.Config, assertions *[]model.Assertion) (agg errorz.Aggregated) {
	var matchingArgs []string
	for _, arg := range args {
		// TODO: IF MATCH
		// Need to concat args which must be concatenated
		_ = arg
	}

	for _, arg := range matchingArgs {
		// TODO: separate operator & value
		_ = arg
		var matchingPrefix, matchingName, matchingOp, matchingValue string
		for _, op := range r.operators {
			if op.op == matchingOp {
				mapper := *op.mapper
				value, err := mapper(matchingOp, matchingValue)
				if err != nil {
					agg.Add(err)
				} else if r.assertion {
					assertion := model.Assertion{
						Rule: model.Rule{
							Prefix:   matchingPrefix,
							Name:     matchingName,
							Op:       matchingOp,
							Expected: matchingValue,
						},
					}
					*assertions = append(*assertions, assertion)
				} else if r.mutater != nil {
					mutater := *r.mutater
					mutater(cfg, matchingOp, value)
				}
			}
		}
	}
	return agg
}

type config struct {
	prefix string
	rule   ruleMatcher
	op     string
	value  string
}

func (c config) Mutate(cfg *model.Config, assertions *[]model.Assertion) {
	// TODO
	return
}

type config2[T any] struct {
	prefix      string
	rule        *rule[T]
	op          string
	value       string
	mappedValue T
}

func (c config2[T]) Mutate(cfg *model.Config, assertions *[]model.Assertion) {
	mutater := *c.rule.mutater
	mutater(cfg, c.op, c.mappedValue)
}

type ruleSet struct {
	name              string
	mutuallyExclusive bool
	dependsOn         []ruleMatcher
	defaults          []ruleMatcher
	rules             []ruleMatcher
	//ruleSets          []RuleSet
}

type mapper[T any] func(op, value string) (T, error)
type validater[T any] func(value T) error
type configMutater[T any] func(cfg *model.Config, op string, value T)

func rules(rules ...ruleMatcher) []ruleMatcher {
	return rules
}

func ops[T any](ops ...*operator[T]) []*operator[T] {
	return ops
}

func buildRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		multiValued: false,
	}
}

func buildMvRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		multiValued: true,
	}
}

func buildAssertRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		multiValued: false,
		assertion:   true,
	}
}

func buildMvAssertRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		multiValued: true,
		assertion:   true,
	}
}

func mutater[T any](f func(*model.Config, string, T)) *configMutater[T] {
	var res configMutater[T] = func(cfg *model.Config, op string, val T) {
		//TODO
	}
	return &res
}

func buildMERS(name string, dependsOn []ruleMatcher, r []ruleMatcher, def ruleMatcher) *ruleSet {
	return &ruleSet{
		name:              name,
		mutuallyExclusive: true,
		dependsOn:         dependsOn,
		defaults:          rules(def),
		rules:             r,
	}
}

func buildRS(name string, dependsOn []ruleMatcher, r []ruleMatcher, def []ruleMatcher) *ruleSet {
	return &ruleSet{
		name:              name,
		mutuallyExclusive: false,
		dependsOn:         dependsOn,
		defaults:          def,
		rules:             r,
	}
}
