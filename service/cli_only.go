package service

import (
	"fmt"

	"mby.fr/cmdtest/asyncdisplay"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/utilz"
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
		Dpl.ClearSuite(ctx)
		Dpl.OpenSuite(ctx)
		Dpl.SuiteTitle(ctx)
	} else {
		// On async init do not display but attempt to clear session
		// asyncDpl := asyncdisplay.NewWaitingTailer(ctx.Repo.BackingFilepath(), false, printz.NewStandardOutputs())
		// asyncDpl.ClearSuite(ctx)
		asyncdisplay.ClearSuite(ctx.Repo.BackingFilepath(), ctx.Config.TestSuite.Get())
	}
	logger.Debug("cleared suite cli side", "suite", ctx.Config.TestSuite.Get(), "async", cfg.Async.Get())

	// Can erase previous suite if it exists
	err = ctx.InitSuite()
	ProcessSuiteError(ctx, err)

	return
}
