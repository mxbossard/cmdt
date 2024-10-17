package display

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"mby.fr/cmdtest/facade"
	"mby.fr/utils/ansi"
	"mby.fr/utils/printz"
)

func NormalizeDurationInSec(d time.Duration) (duration string) {
	duration = fmt.Sprintf("%.3f s", float32(d.Milliseconds())/1000)
	return
}

func TestQualifiedName(ctx facade.TestContext, color ansi.Color) (name string) {
	cfg := ctx.Config
	var testName string
	if cfg.TestName.IsPresent() && !cfg.TestName.Is("") {
		testName = cfg.TestName.Get()
	} else {
		testName = CmdTitle(ctx)
	}

	containerLabel := printz.NewAnsi(TestColor, "on host")
	if ctx.ContainerImage != "" {
		containerLabel = printz.NewAnsi(WarningColor, ctx.ContainerImage)
	}

	name = printz.SColoredPrintf(color, "[%s](%s)>%s", cfg.TestSuite.Get(), containerLabel, testName)
	return
}

func CmdTitle(ctx facade.TestContext) string {
	cmd := ctx.CmdExec
	cmdNameParts := strings.Split(cmd.String(), " ")
	shortenedCmd := filepath.Base(cmdNameParts[0])
	shortenCmdNameParts := cmdNameParts
	shortenCmdNameParts[0] = shortenedCmd
	cmdName := strings.Join(shortenCmdNameParts, " ")
	//testName = fmt.Sprintf("cmd: <|%s|>", cmdName)
	//testName := fmt.Sprintf("[%s]", cmdName)
	testName := cmdName
	return testName
}
