package model

import (
	"regexp"
	"strings"

	"mby.fr/utils/utilz"
)

func NewContext(token string, action Action, cfg Config) Context {
	c := Context{
		Token:  token,
		Action: action,
	}
	c.SetRulePrefix(cfg.Prefix.Get())
	return c
}

type Context struct {
	Token                string
	Action               Action
	rulePrefix           string         // TODEL ?
	assertionRulePattern *regexp.Regexp // TODEL

	Config Config

	TestOutcome  utilz.AnyOptional[TestOutcome]
	SuiteOutcome utilz.AnyOptional[SuiteOutcome]
}

func (c Context) TestId() (id string) {
	// TODO
	return
}

func (c Context) TestQualifiedName() (name string) {
	// TODO
	// qulifiedName := testName
	// if testSuite != "" {
	// 	qulifiedName = fmt.Sprintf("[%s]/%s", testSuite, testName)
	// }
	return
}

func (c Context) TestTitle() (title string) {
	// TODO
	//title = fmt.Sprintf("[%05d] Test %s #%02d", timecode, qualifiedName, seq)
	return
}

func (c Context) RulePrefix() string {
	return c.rulePrefix
}

func (c Context) SetRulePrefix(prefix string) {
	if prefix != "" {
		c.rulePrefix = prefix
		c.assertionRulePattern = regexp.MustCompile("^" + c.RulePrefix() + "([a-zA-Z]+)([=~:!]{1,2})?(.+)?$")
	}
}

func (c Context) IsRule(s string) bool {
	return strings.HasPrefix(s, c.RulePrefix())
}

func (c Context) SplitRuleExpr(ruleExpr string) (ok bool, r Rule) {
	ok = false
	submatch := c.assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		r.Prefix = c.RulePrefix()
		r.Name = submatch[1]
		r.Op = submatch[2]
		r.Expected = submatch[3]
	}
	return
}
