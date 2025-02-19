package parser

import (
	"fmt"
	"regexp"
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
	rules    []ruleMatcher
	ruleSets []*ruleSet
}

func (r *ruleRepo) addRuleSet(rs *ruleSet) {
	// TODO: check for doublons
	// TODO: check for authorized operator
	r.ruleSets = append(r.ruleSets, rs)
	for _, rule := range rs.rules {
		if rule != nil {
			r.rules = append(r.rules, rule)
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

func (r ruleRepo) parseArgs0(prefix string, args []string) (allMatches []ruleMatch, cmdAndArgs []string, agg errorz.Aggregated) {
	// FIXME: add change prefix management
	// FIXME: if ruleParsingStopper encountered consider all following args as cmdAndArgs
	// FIXME: if not matched begin with prefix, do not consider it a cmdAndArgs, but an unkown rule
	args = concatArgs(prefix, args)
	parseRules := true
	for p := 0; p < len(args); p++ {
		if parseRules {
			var replaced bool
			replaced, prefix = prefixReplacer(prefix, args[p])
			if replaced {
				continue
			}
		}

		ruleParsingStopper := prefix + "--"
		if parseRules && args[p] == ruleParsingStopper {
			// Reached rule parsing stopper
			if len(cmdAndArgs) > 0 {
				err := fmt.Errorf("bad placement for command: [%s] before rule parsing stopper %s", cmdAndArgs, ruleParsingStopper)
				agg.Add(err)
			}
			// stop parsing rules
			parseRules = false
			continue
		}

		var matched bool
		if parseRules {
			for _, rs := range r.ruleSets {
				matches, noMatches, agg2 := rs.Match(prefix, args[p:])
				_ = noMatches
				if agg2.GotError() {
					agg.Concat(agg2)
					continue
				}
				if len(matches) == 0 {
					continue
				}
				agg2 = rs.Check(matches...)
				agg.Concat(agg2)

				allMatches = append(allMatches, matches...)
				// remove matched args
				p += len(matches) - 1
				matched = true
			}

			/*
				for _, rule := range r.rules {
					n, configurer, agg2 := rule.Match(prefix, args[p:])
					if agg2.GotError() {
						agg.Concat(agg2)
						continue
					}
					if n == 0 {
						continue
					}
					configurers = append(configurers, configurer)
					// remove matched args
					p += n - 1
					matched = true
				}
			*/
		}

		if !matched && (!strings.HasPrefix(args[p], prefix) || !parseRules) {
			cmdAndArgs = append(cmdAndArgs, args[p])
		} else if !matched {
			agg.Add(fmt.Errorf("unkown rule: [%s]", args[p]))
		}

		for _, rs := range r.ruleSets {
			agg2 := rs.Check(allMatches...)
			agg.Concat(agg2)
		}
	}

	return
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

func ParseArgs(args []string) (cfg model.Config, err error) {
	return
}

func buildArgsConfig(prefix string, args []string) (configs []config0, err error) {
	//TODO
	return
}

func completeArgsConfig(prefix string, args []string) (configs []config0, err error) {
	//TODO
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
