package service

import (
	"fmt"

	"cmdt/internal/asyncdisplay"
	"cmdt/internal/display"
	"cmdt/internal/facade"
	"cmdt/internal/model"
	"cmdt/internal/utils"

	"github.com/mxbossard/utilz/utilz"
)

func cliInitTestSuite(ctx facade.SuiteContext) (exitCode int16, err error) {
	logger.Debug("Initializing test suite", "token", ctx.Token, "isolation", ctx.Isolation, "suites", ctx.Config.TestSuite)
	// Clear and Init new test suite
	exitCode = 0
	cfg := ctx.Config

	var token string
	if cfg.PrintToken.Is(true) {
		token, err = utils.ForgeUuid()
		if err != nil {
			return
		}
		logger.Debug("printToken", "token", ctx.Token)
		fmt.Printf("%s\n", token)
		cfg.Token = utilz.OptionalOf(token)
	} else if cfg.ExportToken.Is(true) {
		token, err = utils.ForgeUuid()
		if err != nil {
			return
		}
		logger.Debug("exportToken", "token", ctx.Token)
		fmt.Printf("export %s=%s\n", model.ContextTokenEnvVarName, token)
		cfg.Token = utilz.OptionalOf(token)
	}

	if !cfg.Async.Get() {
		Dpl.ClearSuite(cfg.TestSuite.Get())
		Dpl.OpenSuite(ctx)
		Dpl.SuiteTitle(ctx)
	} else {
		// On async init do not display but attempt to clear session
		asyncdisplay.ClearSuite(ctx.Repo.BackingFilepath(), ctx.Config.TestSuite.Get())
	}
	logger.Debug("cleared suite cli side", "suite", ctx.Config.TestSuite.Get(), "async", cfg.Async.Get())

	// Can erase previous suite if it exists
	err = ctx.InitSuite()
	ProcessSuiteError(ctx, err)

	return
}

func cliAfterSuiteReport(token, isolation, suite string, dpl display.Displayer) (err error) {
	rep := facade.Repo(token, isolation)
	defer rep.PoolClose()
	err = rep.MarkSuiteReported(suite)

	if err != nil {
		return
	}
	dpl.ClearSuite(suite)
	return
}
