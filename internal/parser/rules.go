package parser

import (
	"fmt"
	"slices"
	"strings"

	"cmdt/internal/model"
	"github.com/mxbossard/utilz/collectionz"
	"github.com/mxbossard/utilz/errorz"
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

type RuleDef interface {
	Name() string
	Kind() string
	Aliases() []string
	Ops() []string
	Matcher() ruleMatcher
}

type ruleMatch interface {
	Prefix() string
	Name() string
	Op() string
	Def() RuleDef
	Mutate(cfg *model.Config, assertions *[]model.Assertion)
}

type matchChecker interface {
	Name() string
	Check(...ruleMatch) errorz.Aggregated
}

type ruleMatcher interface {
	RuleDef
	matchChecker
	Name() string
	Aliases() []string
	Match(prefix string, args []string) (int, ruleMatch, errorz.Aggregated)
	//MutateIfMatch(args []string, cfg *model.Config, assertions *[]model.Assertion) errorz.Aggregated
}

type ruleSetMatcher interface {
	matchChecker
	Match(prefix string, args []string) ([]ruleMatch, []string, errorz.Aggregated)
}

type operator[T any] struct {
	op         string
	mapper     *mapper[T]
	validaters []*validater[T]
}

type rule[T any] struct {
	name           string
	kind           string
	operators      []*operator[T]
	mutater        *configMutater[T]
	aliases        []string
	prefixMask     string // @: Standard behavior: rules prefixed by PREFIX ; -: allow - and -- ;
	assertion      bool
	multiValued    bool
	multiValuedOps []*operator[T]
	exclusiveOps   []*operator[T]
}

func (r rule[T]) Name() string {
	return r.name
}

func (r rule[T]) Kind() string {
	return r.kind
}

func (r rule[T]) Aliases() (a []string) {
	a = append(a, r.Name())
	a = append(a, r.aliases...)
	return
}

func (r rule[T]) Ops() []string {
	return collectionz.Map[*operator[T], string](&r.operators, func(op *operator[T]) string {
		return op.op
	})
}

func (r rule[T]) Matcher() ruleMatcher {
	return r
}

func (r rule[T]) Check(matches ...ruleMatch) (agg errorz.Aggregated) {
	matchingCount := 0
	countByOperatorMap := make(map[string]int, 2)
	ruleByOperatorMap := make(map[string][]string, 2)
	var matchingRules []string // FIXME: could be a set
	for _, match := range matches {
		if r.name == match.Name() || slices.Contains(r.aliases, match.Name()) {
			// Can check against this match
			matchingCount++
			countByOp := countByOperatorMap[match.Op()] + 1
			countByOperatorMap[match.Op()] = countByOp
			ruleByOperatorMap[match.Op()] = append(ruleByOperatorMap[match.Op()], match.Prefix()+match.Name()+match.Op())
			matchingRules = append(matchingRules, match.Prefix()+match.Name()+match.Op())
		}
	}
	matchingRules = collectionz.Deduplicate(&matchingRules)
	uniqOps := collectionz.Sub(&r.operators, &r.multiValuedOps)
	for _, op := range uniqOps {
		if countByOperatorMap[op.op] > 1 {
			usedRules := ruleByOperatorMap[op.op]
			usedRules = collectionz.Deduplicate(&usedRules)
			err := fmt.Errorf("rule [%s%s] is uniq and cannot be used more than once (like: [%s])", r.name, op.op, strings.Join(usedRules, ", "))
			agg.Add(err)
		}
	}
	for _, op := range r.exclusiveOps {
		exclusiveCount := countByOperatorMap[op.op]
		if exclusiveCount > 0 && matchingCount > exclusiveCount {
			otherOps := collectionz.Delete(matchingRules, op.op)
			err := fmt.Errorf("rule [%s%s] is exclusive and cannot be used with other operators (like: [%s])", r.name, op.op, strings.Join(otherOps, ", "))
			agg.Add(err)
		}
	}
	return
}

func (r rule[T]) Match(prefix string, args []string) (n int, match ruleMatch, agg errorz.Aggregated) {
	// Verify if supplied args match the rule
	// If so, return the args count matched, the errors encountered and an object able to mutate the config
	// FIXME: should return an object able to mutate the config

	if len(args) == 0 {
		//fmt.Printf("no args\n")
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
		//fmt.Printf("no prefix matching\n")
		return 0, nil, agg
	}

	// 2- Determinate which alias is used
	var matchingAlias string
	var matchingRule string
	if strings.HasPrefix(args[0], prefix+r.name) {
		matchingAlias = r.name
		matchingRule = prefix + r.name
	} else {
		for _, alias := range r.aliases {
			if strings.HasPrefix(args[0], prefix+alias) {
				matchingAlias = alias
				matchingRule = prefix + alias
				break
			}
		}
	}

	if matchingRule == "" {
		// Does not match the rule
		return 0, nil, agg
	}

	// 3- identifiy if rule name and operator match
	if r.operators == nil {
		// To match must not have an operator nor a value
		if args[0] == matchingRule {
			n = 1
			cfg2 := &basicRuleMatch[T]{}
			cfg2.op = ""
			cfg2.prefix = prefix
			cfg2.rule = &r
			cfg2.value = ""
			match = cfg2
			//fmt.Printf("match no operator\n")
			return
		}
	}

	matchingOpLen := -1
	var matchingOp *operator[T]
	var value string
	for _, op := range r.operators {
		// Multiple op could match, must keep longest op
		if op.op == " " {
			// Special case: space operator
			if args[0] == matchingRule {
				if len(args) == 1 {
					n = 1
					agg.Add(fmt.Errorf("missing arg after space operator"))
					//fmt.Printf("missing arg after space operator\n")
					return
				}
				matchingOp = op
				n = 2
				value = args[1]
			}
			break
		}
		//fmt.Printf("testing if arg: [%s] match: [%s]\n", args[0], matchingRule+op.op)
		if len(op.op) > matchingOpLen && ((op.op == "" && args[0] == matchingRule+op.op) ||
			(op.op != "" && strings.HasPrefix(args[0], matchingRule+op.op))) {
			// if longer op AND ( (noOp + match exactly) OR (op + match begining) )
			//fmt.Printf("op: [%s] match\n", op.op)
			matchingOpLen = len(op.op)
			matchingOp = op
			n = 1
			value = strings.TrimPrefix(args[0], matchingRule+op.op)
		}
	}

	if matchingOp == nil {
		// FIXME: if one alias match but no operator should hint with an error
		//fmt.Printf("no operator match\n")

		p := len(matchingRule)
		if len(args[0]) > p && strings.HasPrefix(args[0], matchingRule) {
			// prefix & rule name match
			badOp := operatorPrefix(args[0][p:])
			if badOp != "" {
				//  valid operator not registered for the rule is used
				agg.Add(fmt.Errorf("invalid operator: [%s] used for rule: [%s] (%s)", badOp, matchingRule, args[0]))
			}
		}

		return 0, nil, agg
	}

	// 4- validate value
	var mappedValue T
	var err error
	if matchingOp.mapper != nil {
		mapper := *matchingOp.mapper
		mappedValue, err = mapper(matchingOp.op, value)
		if err != nil {
			agg.Add(fmt.Errorf("invalid value for rule [%s%s%s%s], error: %w", prefix, matchingAlias, matchingOp.op, value, err))
		}
	} else {
		// no mapper => no value : nothing to do
		/*
			var val any = value
			var ok bool
			mappedValue, ok = val.(T)
			if !ok {
				panic(fmt.Errorf("unable to map value of rule: [%s]", r.name))
			}
		*/
	}

	for _, validaterPtr := range matchingOp.validaters {
		validater := *validaterPtr
		err := validater(mappedValue)
		if err != nil {
			agg.Add(err)
		}
	}

	if agg.GotError() {
		//fmt.Printf("some error\n")
		return n, nil, agg
	}
	cfg2 := &basicRuleMatch[T]{}
	cfg2.prefix = prefix
	cfg2.name = matchingAlias
	cfg2.op = matchingOp.op
	cfg2.rule = &r
	cfg2.value = value
	cfg2.mappedValue = mappedValue
	match = cfg2
	//fmt.Printf("match\n")
	return
}

type basicRuleMatch[T any] struct {
	rule        *rule[T]
	prefix      string
	name        string
	op          string
	value       string
	mappedValue T
}

func (c basicRuleMatch[T]) Prefix() string {
	return c.prefix
}

func (c basicRuleMatch[T]) Name() string {
	return c.name
}

func (c basicRuleMatch[T]) Op() string {
	return c.op
}

func (c basicRuleMatch[T]) Def() RuleDef {
	return c.rule
}

func (c basicRuleMatch[T]) Mutate(cfg *model.Config, assertions *[]model.Assertion) {
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

func (r ruleSet) Name() string {
	return r.name
}

func (r ruleSet) Aliases() []string {
	return collectionz.Flatten(collectionz.Map(&r.rules, func(r ruleMatcher) []string {
		return r.Aliases()
	})...)
}

func (r ruleSet) Contains(rd RuleDef) bool {
	for _, rule := range r.rules {
		if rule.Name() == rd.Name() {
			return true
		}
	}
	return false
}

func (r ruleSet) Check(matches ...ruleMatch) (agg errorz.Aggregated) {
	// TODO: perform cfgs valdiation for each rule
	// FIXME: how to delegate validation of each rule to each rule ?
	for _, rule := range r.rules {
		// Check matches agains each rule
		agg2 := rule.Check(matches...)
		agg.Concat(agg2)
	}

	// TODO: perform ruleSet validations
	// FIXME: need to validate against aliases
	if r.mutuallyExclusive {
		// Check for Mutually exclusive usage
		var ruleAliases []string
		for _, rm := range r.rules {
			ruleAliases = append(ruleAliases, rm.Name())
			ruleAliases = append(ruleAliases, rm.Aliases()...)
		}

		var matchedRules []string // FIXME: could be a set
		for _, match := range matches {
			//fmt.Printf("Does %s is in: %s ?\n", match.Name(), ruleAliases)
			if slices.Contains(ruleAliases, match.Name()) {
				matchedRules = append(matchedRules, match.Name())
			}
		}
		matchedRules = collectionz.Deduplicate(&matchedRules)
		if len(matchedRules) > 1 {
			err := fmt.Errorf("[%s] rules are mutually exclusives", strings.Join(matchedRules, ", "))
			agg.Add(err)
		}
	}

	return
}

func (r ruleSet) Match(prefix string, args []string) (matches []ruleMatch, notMatched []string, agg errorz.Aggregated) {
	// TODO: Match each args agains each rules MUST manage prefix changes and "stop parsing"
	// TODO: perform rules multiValuedOps & eclusiveOps validations
	// TODO: perform ruleSet validations

	args = concatArgs(prefix, args)
	parseRules := true
	for p := 0; p < len(args); p++ {
		if parseRules {
			var replaced bool
			replaced, prefix = prefixReplacer(prefix, args[p])
			if replaced {
				notMatched = append(notMatched, args[p])
				continue
			}
		}

		ruleParsingStopper := prefix + "--"
		if parseRules && args[p] == ruleParsingStopper {
			// Reached rule parsing stopper
			/*
				if len(cmdAndArgs) > 0 {
					err := fmt.Errorf("bad placement for command: [%s] before rule parsing stopper %s", cmdAndArgs, ruleParsingStopper)
					agg.Add(err)
				}
			*/
			// stop parsing rules
			parseRules = false
			notMatched = append(notMatched, args[p])
			continue
		}

		// TODO: check for prefix change
		// TODO: check for rule stopper
		var matched bool
		if parseRules {
			for _, rule := range r.rules {
				n, match, agg2 := rule.Match(prefix, args[p:])
				agg.Concat(agg2)
				if n > 0 {
					// matched
					matched = true
					p += n - 1
					if match != nil {
						matches = append(matches, match)
					}
				}
			}
		}
		if !matched {
			notMatched = append(notMatched, args[p])
		}
	}

	/*
		for _, rule := range r.rules {
			agg2 := rule.Check(matches...)
			agg.Concat(agg2)
		}

		agg2 := r.Check(matches...)
		agg.Concat(agg2)
	*/
	return
}

type metaRuleSet struct {
	name            string
	ruleSetMatchers []ruleSetMatcher
}

func (r metaRuleSet) Name() string {
	return r.name
}

func (r metaRuleSet) Match(prefix string, args []string) (matches []ruleMatch, notMatched []string, agg errorz.Aggregated) {
	// TODO: Match each args against each ruleSetMatcher
	notMatched = args
	for _, rsm := range r.ruleSetMatchers {
		matches2, notMatched2, agg2 := rsm.Match(prefix, notMatched)
		matches = append(matches, matches2...)
		notMatched = notMatched2
		agg.Concat(agg2)
	}
	// TODO: perform metaRuleSet validations
	return
}

type mapper[T any] func(op, value string) (T, error)
type validater[T any] func(value T) error
type configMutater[T any] func(cfg *model.Config, op string, value T)

func rules(rules ...ruleMatcher) []ruleMatcher {
	return rules
}

func ruleDefs(rules ...ruleMatcher) []RuleDef {
	return defs(rules)
}

func defs(rules []ruleMatcher) (defs []RuleDef) {
	for _, rule := range rules {
		defs = append(defs, rule)
	}
	return
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
		prefixMask:  "@",
		multiValued: false,
	}
}

func buildMvRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:           name,
		operators:      ops,
		mutater:        mutater,
		aliases:        aliases,
		prefixMask:     "@",
		multiValued:    true,
		multiValuedOps: ops,
	}
}

func buildAssertRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		prefixMask:  "@",
		multiValued: false,
		assertion:   true,
	}
}

func buildMvAssertRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:           name,
		operators:      ops,
		mutater:        mutater,
		aliases:        aliases,
		prefixMask:     "@",
		multiValued:    true,
		assertion:      true,
		multiValuedOps: ops,
	}
}

func buildMvExclOpsAssertRule[T any](name string, ops []*operator[T], mvOps []*operator[T], exclOps []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:           name,
		operators:      ops,
		mutater:        mutater,
		aliases:        aliases,
		prefixMask:     "@",
		multiValued:    true,
		assertion:      true,
		multiValuedOps: mvOps,
		exclusiveOps:   exclOps,
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
