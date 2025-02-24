package parser

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/collections"
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
			allNoMatchesIntersect = collections.Intersect(&allNoMatchesIntersect, &allNoMatches[p])
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

	return
}

func (r ruleRepo) childs(args ...string) (ruleDefs []RuleDef, warns errorz.Aggregated, errors errorz.Aggregated) {
	// TODO implements warnings & errors

	var lastRuleDef RuleDef
	for _, arg := range args {
		argRuleName := extractRuleName(arg)
		if argRuleName == "" {
			errors.Add(fmt.Errorf("unknown rule: [%s]", arg))
			return
		}
	exit:
		for _, ruleSet := range r.ruleSets {
			for _, rule := range ruleSet.rules {
				if argRuleName == arg || slices.Contains(rule.Aliases(), argRuleName) {
					// arg matches the rule
					lastRuleDef = rule
					break exit
				}
			}
		}
	}

	var dependingSets []*ruleSet
	for _, ruleSet := range r.ruleSets {
		if len(args) == 0 && len(ruleSet.dependsOn) == 0 {
			// should select all sets depending on nothing
			dependingSets = append(dependingSets, ruleSet)
		} else if slices.Contains(defs(ruleSet.dependsOn), lastRuleDef) {
			// FIXME: the following test probably don't works for ptr reason
			// Last arg depens on this rule
			dependingSets = append(dependingSets, ruleSet)
		}
	}

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
	assertionRulePattern := regexp.MustCompile("^(?:@|-|--)?([-_a-zA-Z]+).*")
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
