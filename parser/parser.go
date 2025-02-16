package parser

import (
	"fmt"
	"regexp"
	"strings"

	"mby.fr/cmdtest/model"
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

func (r ruleRepo) parseArgs(prefix string, args []string) (configurers []configurer, agg errorz.Aggregated) {
	args = concatArgs(prefix, args)
	for len(args) > 0 {
		var matched bool
		for _, rule := range r.rules {
			n, configurer, agg2 := rule.Match(prefix, args)
			if agg2.GotError() {
				agg.Concat(agg2)
				continue
			}
			if n == 0 {
				continue
			}
			configurers = append(configurers, configurer)
			// remove matched args
			args = args[n:]
			matched = true
		}

		if !matched {
			agg.Add(fmt.Errorf("unable to parse args: [%s]", args))
			return
		}
	}

	return
}

func ParseArgs(args []string) (cfg model.Config, err error) {
	return
}

func buildArgsConfig(prefix string, args []string) (configs []config, err error) {
	//TODO
	return
}

func completeArgsConfig(prefix string, args []string) (configs []config, err error) {
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
