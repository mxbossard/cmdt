package parser

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/collectionz"
	"mby.fr/utils/errorz"
	"mby.fr/utils/zlog"
)

var (
	logger = zlog.New()
)

type ruleRepo struct {
	rules    []RuleDef
	ruleSets []*ruleSet
}

func (r *ruleRepo) addRuleSet(rs *ruleSet) {
	// TODO: check for doublons
	// TODO: check for authorized operator
	r.ruleSets = append(r.ruleSets, rs)
	for _, rule := range rs.rules {
		if rule != nil {
			var rd RuleDef
			rd = rule
			r.rules = append(r.rules, rd)
		}
	}
}

func prefixReplacer(prefix, rule string) (bool, string) {
	prefixPattern := regexp.MustCompile("^" + prefix + "prefix=(.)$")
	submatch := prefixPattern.FindStringSubmatch(rule)
	if submatch != nil {
		return true, submatch[1]
	}
	return false, prefix
}

func (r ruleRepo) parseArgs(prefix string, args []string) (allMatches []ruleMatch, cmdAndArgs []string, agg errorz.Aggregated) {
	args = concatArgs(prefix, args)

	firstRuleParsingStopper := ""
	parseRuleStopperPos := -1
	prefix2 := prefix
	for p, arg := range args {
		replaced := false
		replaced, prefix2 = prefixReplacer(prefix2, arg)
		if replaced {
			continue
		}
		ruleParsingStopper := prefix2 + "--"
		if arg == ruleParsingStopper {
			if parseRuleStopperPos > -1 {
				err := fmt.Errorf("rule parsing stopper: [%s] must be uniq", ruleParsingStopper)
				agg.Add(err)
			} else {
				parseRuleStopperPos = p
				firstRuleParsingStopper = ruleParsingStopper
			}
		}
	}

	if parseRuleStopperPos > -1 {
		cmdAndArgs = args[parseRuleStopperPos+1:]
		args = args[0:parseRuleStopperPos]
	}

	var allNoMatches [][]string
	for _, rs := range r.ruleSets {
		matches, noMatches, agg2 := rs.Match(prefix, args)
		agg.Concat(agg2)

		//fmt.Printf("Checking(RS=%s) %s for args: %s...\n", rs.name, matches, args)
		agg2 = rs.Check(matches...)
		//fmt.Printf("Agg2 errors: %s\n", agg2)
		agg.Concat(agg2)

		allMatches = append(allMatches, matches...)
		allNoMatches = append(allNoMatches, noMatches)
	}

	allNoMatchesIntersect := allNoMatches[0]
	if len(allNoMatches) > 1 {
		for p := 1; p < len(allNoMatches); p++ {
			allNoMatchesIntersect = collectionz.Intersect(&allNoMatchesIntersect, &allNoMatches[p])
		}
	}

	prefix2 = prefix
	for _, noMatch := range allNoMatchesIntersect {
		replaced := false
		replaced, prefix2 = prefixReplacer(prefix2, noMatch)
		if replaced {
			continue
		}

		if !strings.HasPrefix(noMatch, prefix) && parseRuleStopperPos > -1 {
			// noMatch arg without prefix is encountered before rule parsing stopper
			if len(cmdAndArgs) > 0 {
				err := fmt.Errorf("bad placement for command: [%s] before rule parsing stopper %s", cmdAndArgs, firstRuleParsingStopper)
				agg.Add(err)
			}
		} else if !strings.HasPrefix(noMatch, prefix2) {
			// noMatch arg without prefix is encountered
			cmdAndArgs = append(cmdAndArgs, noMatch)
		} else {
			// noMatch arg with prefix is encountered
			agg.Add(fmt.Errorf("unkown rule: [%s]", noMatch))
		}

	}

	// FIXME: Validate each rule found satisfy it's dependency
	// 1- foreach rule get ruleSet
	// 2- if ruleset depends on a rule, check the rule is present or add error
	allMatchesNames := collectionz.Map(&allMatches, func(rm ruleMatch) string {
		return rm.Name()
	})
Exit:
	for _, rule := range allMatches {
		var containingRuleSet *ruleSet
		for _, rs := range r.ruleSets {
			if rs.Contains(rule.Def()) {
				containingRuleSet = rs
				break
			}
		}
		if containingRuleSet != nil && len(containingRuleSet.dependsOn) > 0 {
			// Verify at least one of depending rule is present
			dependendingNames := collectionz.Map(&(containingRuleSet.dependsOn), func(rm ruleMatcher) string {
				return rm.Name()
			})
			for _, dependendingName := range dependendingNames {
				if slices.Contains(allMatchesNames, dependendingName) {
					break Exit
				}
			}
			agg.Add(fmt.Errorf("rule: [%s] can only be used in context of rules: [%s]", rule.Name(), strings.Join(dependendingNames, ", ")))
		}
	}

	return
}

func (r ruleRepo) children(args ...string) (ruleDefs []RuleDef, warns errorz.Aggregated, errors errorz.Aggregated) {
	// 1- Identify the rule targeted by the supplied args (path)
	// 1a- Validate path to targeted rule
	// 2- List rules depending on the targeted rule + not dependings rules (?? isolate this in global() func ??)
	// 2a- Scope depending rules to supplied path ?
	// 2b- which not depending rules add ? config ok but not actions !

	prefixes := []string{"@", ""}

	// 1- Identify the last RuleDef in args path
	var lastRuleDef RuleDef
	var lastRuleChilrenNames []string
	for p, arg := range args {
		argRuleName := extractRuleName(arg)
		if argRuleName == "" {
			errors.Add(fmt.Errorf("malformed rule: [%s]", arg))
			return
		}
		// FIXME: args path is not checked
		var knownRule bool
	Exit:
		for _, ruleSet := range r.ruleSets {
			for _, rule := range ruleSet.rules {
				//fmt.Printf("Checking arg: %s against aliases: %s \n", arg, rule.Aliases())
				if slices.Contains(rule.Aliases(), argRuleName) {
					// arg matches the rule
					//fmt.Printf(">> arg: %s match rule name: %s \n", arg, rule.Name())
					knownRule = true
					lastRuleDef = rule
					break Exit
				}
			}
		}

		if !knownRule {
			// arg not found in any ruleset
			errors.Add(fmt.Errorf("unknown rule: [%s]", arg))
			return
		}

		if p > 0 {
			// Validate path: is rule a child of previous rule
			if !slices.Contains(lastRuleChilrenNames, lastRuleDef.Name()) {
				errors.Add(fmt.Errorf("rule: [%s] is not a child of rule: [%s] (%s)", args[p], args[p-1], lastRuleChilrenNames))
			}
		}

		// List rule children to validate next rule ancestry
		if len(args) > 1 && p < len(args) {
			lastRuleChilren, ws, es := r.children(args[p])
			lastRuleChilrenNames = collectionz.Map(&lastRuleChilren, func(r RuleDef) string { return r.Name() })
			warns.Concat(ws)
			errors.Concat(es)
		}
	}

	// 2- Attempt to match the last RuleDef with the minimum last supplied args
	if lastRuleDef != nil {
		lastRuleMatcher := lastRuleDef.Matcher()
		var lastRuleMatch ruleMatch
		l := len(args)
	Exit2:
		for _, prefix := range prefixes {
			// Attempt to match with last arg, then with 2 last args, ...
			for p := 1; p <= l; p++ {
				lastArgs := args[l-p : l]
				//fmt.Printf(">> matching args: %s with rule: %s ... \n", lastArgs, lastRuleMatcher.Name())
				n, m, agg := lastRuleMatcher.Match(prefix, lastArgs)
				warns.Concat(agg)
				if n > 0 {
					// found args matched by the last rule
					lastRuleMatch = m

					break Exit2
				}
			}
		}

		// if lastRuleMatch == nil {
		// 	errors.Add(fmt.Errorf("problem with not matching rule: [%s]", lastRuleDef))
		// }
		if lastRuleMatch != nil {
			agg := lastRuleMatcher.Check(lastRuleMatch)
			warns.Concat(agg)
		}
	}

	// 3- List ruleSets dependings on last rule
	var dependingSets []*ruleSet
	for _, ruleSet := range r.ruleSets {
		dependsOnNames := collectionz.Map(&ruleSet.dependsOn, func(r ruleMatcher) string {
			return r.Name()
		})
		if len(args) == 0 && len(ruleSet.dependsOn) == 0 {
			// should select all sets depending on nothing
			dependingSets = append(dependingSets, ruleSet)
		} else if lastRuleDef != nil && slices.Contains(dependsOnNames, lastRuleDef.Name()) {
			// Last arg depends on this rule
			//fmt.Printf("ruleSet: %s depends on args: [%s]\n", ruleSet.Name(), args)
			dependingSets = append(dependingSets, ruleSet)
		}
	}

	/*
		dependingSetsNames := collections.Map(&dependingSets, func(r *ruleSet) string {
			return r.Name()
		})
		if lastRuleDef != nil {
			fmt.Printf("rule: %s depends on sets: [%s]\n", lastRuleDef.Name(), dependingSetsNames)
		} else {
			fmt.Printf("rule: [] depends on sets: [%s]\n", dependingSetsNames)
		}
	*/

	// 4- Append rules of dependings ruleSets
	for _, ruleSet := range dependingSets {
		ruleDefs = append(ruleDefs, defs(ruleSet.rules)...)
	}

	return
}

func replacingPrefix(prefix, rule string) string {
	prefixPattern := regexp.MustCompile("^" + prefix + "prefix=(.)$")
	submatch := prefixPattern.FindStringSubmatch(rule)
	if submatch != nil {
		return submatch[1]
	}
	return prefix
}

func extractRuleName(arg string) string {
	assertionRulePattern := regexp.MustCompile("^(?:@|-|--)?([a-zA-Z][-_a-zA-Z]*).*")
	submatch := assertionRulePattern.FindStringSubmatch(arg)
	if submatch != nil {
		return submatch[1]
	}
	return ""
}

func concatArgs(rulePrefix string, args []string) []string {
	// If a ruleParsingStopper is used, may concat args before ruleParsingStopper

	prefix := rulePrefix
	// Check if ruleParsingStopper is present
	ruleParsingStopperPresent := false
	for _, arg := range args {
		prefix = replacingPrefix(prefix, arg)
		ruleParsingStopper := prefix + model.RuleParsingStopper
		if arg == ruleParsingStopper {
			ruleParsingStopperPresent = true
			break
		}
	}

	prefix = rulePrefix
	// Concatenate args with space before ruleParsingStopper
	if ruleParsingStopperPresent {
		var concatenatedArgs []string
		// Check all args before ruleParsingStopper
		var buffer string
		for p, arg := range args {
			prefix = replacingPrefix(prefix, arg)
			ruleParsingStopper := prefix + model.RuleParsingStopper
			if arg == ruleParsingStopper {
				if buffer != "" {
					concatenatedArgs = append(concatenatedArgs, buffer)
				}
				concatenatedArgs = append(concatenatedArgs, args[p:]...)
				break
			} else if model.MatchRuleDef(prefix, arg, model.Concatenables...) {
				// Concatenable rule
				if buffer != "" {
					// flush buffer
					concatenatedArgs = append(concatenatedArgs, buffer)
				}
				// init buffer
				buffer = arg
			} else if strings.HasPrefix(arg, prefix) {
				if buffer != "" {
					// flush buffer
					concatenatedArgs = append(concatenatedArgs, buffer)
				}
				buffer = ""
				concatenatedArgs = append(concatenatedArgs, arg)
			} else if buffer == "" {
				concatenatedArgs = append(concatenatedArgs, arg)
			} else {
				buffer += " " + arg
			}
		}

		logger.Debug("concatenated args", "args", args, "concatenated", concatenatedArgs)
		// replace args by concatenated ones
		return concatenatedArgs
	}
	return args
}
