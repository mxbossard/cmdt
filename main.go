/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"log/slog"
	"os"

	"mby.fr/cmdtest/daemon"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/service"
)

/*

## Fork ideas:
- before/after and dirtiesScope cause problems to run in //
- @tests could always be added in a queue, a background process (daemon) in charge of executing it
- one queue by test suite ?
- group outputs by test suite : first output take output priority (like mass do)
- new test : enqueue test, wait some ms ?, check daemon is started or start it
- no test in queue : wait some secs, stop daemon
- could wait @report before launching tests if necessary ?
- @report could block until all tests done optionnaly ?
- only one dameon running at a time
- new rules @async[=true/false] @block[=true/false] @_daemon
- do not run daemon in container


## TODO:

Bugs:
- Check for container existance before exec in running container
- Suite Timeout not managed (should error if timeout exceeded) Should ask for suite clear and no test should pass; initless suite should have a greater default timeout
- use suite timeout for container duration
- @global config updates does not works
- serialize test outcome instead of writing in report file
- slower podman exec than docker exec
- colors ok on black background but should not work on white background


Cleaning:
- move seq into utils module
- move contianer use into utils module


Optims:
- DONE: stoping container can be done async (no need to block for end of stop)
- mocking container can all be done in //
- start container can be call async but execs need to wait container to be started


Features:
- use rule definitions in usage
- @beforeSuite=CMD_ANG_ARGS & @afterSuite=CMD_ANG_ARGS
- @called[=:]CMD ARG_S,stdin=IN,count=N assertion => verify a mock was called
- probably too slow podman/docker abstraction (check for podman & docker in path everytime)
- docker/podman image pre pull once outside of test timeout (store pull in global context)
- docker/podman image pre build once if supplied ref is a buildable context dir
- docker/podman container engine resolution once stored in global context
- docker/podman checkpoint for dirties to avoid container rm/creation

- rewrite mock wrapper script inside cmdt to not depend on shell to mock
- possibilité de passer un scénario ligne à ligne dans le stdin de cmdtest
	- un scenario pourrait-il être de la forme d'un script shell ? => être executable comme scenario et comme script ?
	- cmdt cmd arg1 argN @scenario=filepath
	- pour chaque ligne du scenario concat la ligne du scenario avec les arguments fournit en paramétre de cmdt
- @testAll[=dirToScan] by default scan current dir in search for scenarios or script to execute (could run .cmdt scenarios & scripts)
- @runCount=N + @parallel=FORK_COUNT (min, max, median exec time) run in separate context or in same context (before/after) guided by @dirtyRun
- @fork=5 suite/global config only by default instead of @parallel. Fork = 5 increment and decrement a seq file
  - what is forked ?
  - by default suites forked but test serial in a suite
  - optionaly all tests forked in a suite
  - fork in a container ? or one container by fork ? (forking implies test independency @dirtyContainer only should guide for new container)
- change default test suite with @init=foo => foo become default test suite

- Clean old temp dir (older than 2 min ?)
- may chroot be interesting for tests ?
- mock web spawning a web server ?
- test port opening if daemon ; test sending data on port ???


## DONE:

- @timeout=N
- @cmd="CMD_WITH_ARGS_THAT_SHOULD_BE_OK_IF_OR_FAIL_TEST"
- @exists=FILEPATH,PERMS,OWNERS
- Improve cmd log delimiting args:
        no spaces: CMD ARG_1 ARG_2 ARG_N
        some spaces: <[ CMD AVEC ESPACE, ARG_1, ARG AVEC ESPACE, ARG_N ]>
        autre idée ??? CMD ARG_1 'ARG 2' ... <|ARG N|>  or 'CMD WITH SPACE' ARG_1 'ARG 2' ... ARG_N Possible separators: simple quotte ; <| |> ;
- pass file content by filepath to @stdout @stderr : @stdout=@FILEPATH
- @prefix= change @ char ??? attention à la sécurité ça pourrait etre galere
- tester cmdt avec cmdt ?
	- cmdt @prefix=% cmdt true %success => should success
	- cmdt @prefix=% cmdt false %fail => should success
	- cmdt @prefix=% cmdt false @fail %success => should success
- TestSuite context resolving
	- DO WE NEED to init a tmpdir to store testsuite context ?
		- multiples test suites share a tmpdir
		- a testsuite can be inited with first test
		- init clear all previous context => easiest context resolution
		- without init may have multiple matching context (same workspace )
	- matching test suite context should be in workspace, touched recently, (and share the ppid ?)
	- for specific use case (if tests does not share ppid or workspace dir) can pass a uniq token to identify a test suite
	    - NEW tk=$( cmdt @init @token ) => print an uniq token on stdout
		- @init=foo @token=$tk => init a foo test suite with tk token identifier
		- @test=foo @token=$tk => test in suite within token tk context
		- @report=foo @token=$tk
		- NEW eval $( @token @export ) => print the uniq token export on stdout to be supplied to all following cmdt call
		- tokened context is uniq accross all workspace dirs (can change workspace or ppid during test)
	- which tmpdir ?
	    - global tmp => to mitigate the risk of test suite mixing should hash the workspace dir path and put it into tmp dir
		- share a tmpdir between multiple test suites => init only tmp dir and report all tmp dir
		- By default build tmp dir with workspace + PPID + TIMESTAMP
		- If workspace dir change or ppid change => can use a token
- @init=testsuite => clear all matching testsuite
- @test=testsuite => find a matching testsuite. If none => init a new one. If muliple => fail
- @report => by default report all testsuites
- New operators != "not match exactly" !~ "not contains"
- New regex matching with new operators :
	- replace ~ used in contains by :
	- ~/PATTERN/FLAGS "contains match regexp"
	- !~/PATTERN/FLAGS "not match regexp"
- @global action for global config (config for all test suites)
- @silent config hide success
- Rules as constants sorted by type in collections => if rule not in collection fail
- leverage rule definitions for Mutual Exclusions
- @mock des appels de commande @mock="CMD ARG_1 ARG_2 ARG_N,stdin=baz,exit=0,stdout=foo,stderr=bar,cmd=CMD ARG_1 ARG_N"
- @before=CMD ARG_1 ARG_2 ... ARG_N => execute CMD before each test
- @after=CMD ARG_1 ARG_2 ... ARG_N => execute CMD after each test
- Total duration if reporting multiple suites
- order reports : report failures then errors at the end of report
- @dirty= Reset l'image à chaque test suite ou à chaque test en option
- Run en option les tests dans un conteneur => plus facile de mocké, meilleur reproductibilité des tests
	- @container=IMAGE : Fournir une image pour tester l'éxecution dans cette image
	- Utiliser une image par défaut pour les cas d'usage simples
- improve printer. Writer stdout & stder with prefix stdout> stderr> in descriptions
- New Config/Context
- Introduce @verbose=N to manage test verbosity
- Introduce @debug=N to log
- rework failure description : hard to read (remove colors ? remove \n ?)
- Replace @silent by @quiet => with @quiet do not loq anything except cmd outputs if enabled with @keepOutputs
- option @failuresLimit=N to controll TOO MUCH FAILURES
- Rework @verbose : Default not verbose lvl: FAILED_ONLY ; Default verbose lvl: PASSED
  - FAILED_ONLY: Display only Failed tests + assertions
  - FAILED_OUTPUTS: Display+ failed stout & stderr
  - PASSED: Display Passed test without assrtions
  - PASSED_OUTPUTS: Display+ passed outputs
- Forbid mock of shell builtin command (we can use type CMD)
- multiple @mock
- Mock les executable avec un chemin absolu dans les conteneur
- Use podman or docker binary
- with @-- report an error if commands before @--
- with @-- auto concatenate args until next delim or @--
- new operators @= et @: to read content from file
- remove docker run container generated ID from stdout
- clean model.go
- @mock stdin@=FILEPATH stdin:PARTIAL_CONTENT stdout@=FILEPATH stderr@=FILEPATH

*/

func RecoverExiting() {
	if r := recover(); r != nil {
		fmt.Printf("Cause of panic ==>> %q\n", r)
		os.Exit(1)
	}
}

func main() {
	//defer RecoverExiting()
	fmt.Printf("Args: %v", os.Args)

	daemon.TakeOver()

	model.LoggerLevel.Set(slog.Level(8 - model.StartDebugLevel*4))
	exitCode := service.ProcessArgs(os.Args)

	os.Exit(exitCode)
}

/*

## Idées pour la commande de mock
    insérer dans le $PATH un repertoire temp binMocked
    Pour chaque commande mockée on place un lien symbolique de la commande mocké vers le mock dans le repertoire binMocked
    il nous faut un programme mock qui consomme tous les args, compte les appels avec les args et répond de la manière dont il est configuré
    mock config :
        Exact call @mock="CMD,ARG_1,ARG_2,ARG_N;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in order not more
        Partial call @mock="CMD,ARG_1,ARG_2,ARG_N,*;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in order then more args
        Contains call @mock:"CMD,ARG_1,ARG_2,ARG_N;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in various order not more
        Default exit code @mock="CMD,*;exit=1"

## Idées pour executer les tests dans un conteneur
  - Une image par défaut busybox like (avec cmdt déjà à l'interieur ? pas forcément nécéssaire sauf pour le speed)
  - En général, il faut monter le binaire cmdt dans le conteneur et optionnelement l'ajouter au PATH
  - cmdt => run cmdt avec exactement les memes args dans un conteneur jetable démarré à l'instant par cmdt
  - avec quel owner démarrer cmdt dans le conteneur => default / option ??
  - Mock une commande absolue possible, mais nécéssite de déplacer la commande original pour la remplacer par un wrapper
  - En option fournir une limite CPU & ram quel valeur par défaut ?
  - En option fournir un user différent dans le conteneur mais interragir avec le tmpDir avec le bon user !
  - Scope ? discard container after suite, test, run (runCount > 1)
    - @cnrScope=none => do not run inside a container
    - @cnrScope=suite => keep suite ctId in suite ctx
	- @cnrScope=test => use a new container for each test
	- @cnrScope=run => use a new container for each test run
  - Before / After scope ? none, suite, test, run
  - Meilleur idée : Before + BeforeSuite + dirties=beforeSuite/afterSuite/beforeTest/afterTest/beforeRun/afterRun
	- DEFAULT: before > run1 > ... > runN > after
	- OPTION1: before1 > run1 > after1 > ... > beforeN > runN > afterN

	- DEFAULT: runCnr > test1 > ... > testN > killCnr (1 cnr by suite)
	- OPTION1: runCnr1 > test1 > killCnr1 > ... runCnrN > testN > killCnrN (1 cnr by test)
	- OPTION2: runCnr11 > test1run1 > killCnr11 > runCnr12 > test1run2 > killCnr12 > ... runCnrNP > testNrunP > killCnrNP (1 cnr by run)

	- DEFAULT: runCnr > before > run1 > ... > runN > after > killCnr
	- OPTION1: runCnr1 > before1 > run1 > after1 > killCnr1 > ... > runCnrN > beforeN > runN > afterN > killCnrN

	- @beforeSuite run on each container start ???
	- @afterSuite run on each container kill ???

    - contextDirty=beforeRun => each run need a new before and a new container
	- contextDirty=afterRun => each run need a new after and a removed container

	- @dirtyRun => mark test run dirty => enforce new before and after for each run

	- @dirtyContainer=beforeRun => mark ctx dirty before each run => enforce new cnr before each run
	- @dirtyContainer=afterRun => mark ctx dirty after each run => enforce cnr kill after each run
	- @dirtyContainer=beforeTest => mark ctx dirty before each test => enforce new cnr before each test
	- @dirtyContainer=afterTest => mark ctx dirty after each test => enforce cnr kill after each test
	- @dirtyContainer=beforeSuite => mark ctx dirty before each suite => enforce new cnr before each suite (DEFAULT ?)
	- @dirtyContainer=afterSuite => mark ctx dirty after each suite => enforce cnr kill after each suite

	- @global @container => by default @dirtyContainer=never => will share a fresh same container between all suites
	- @init @container => by default @dirtyContainer=beforeSuite => will share a fresh container between all tests in a suite
	- @test @container => by default @dirtyContainer=beforeTest => will share a fresh container between all runs of a test

  - Exemples
    @global @container # dirtyContainer=none
	@test true # run in cnr1
	@test true # run in cnr1
	@test @dirtyContainer=beforeTest true # run in cnr2 (destroyded cnr1 before)
	@test true # run in cnr2
	@test @dirtyContainer=afterTest true # run in cnr2 (will destroy cnr2 after)
	@test true # run in cnr3

	@global @container @dirtyContainer=beforeTest
	@test true # run in cnr1
	@test true # run in cnr2
	@test @dirtyContainer=afterTest true # run in cnr2 (cnr2 not destroyed before)
	@test true # run in cnr3

	@global @container @dirtyContainer=beforeTest
	@test true # all run in cnr1
	@test @dirtyContainer=afterRun true # run1 in cnr1, run2 in cnr2, ..., runN in cnrN
	@test true # all run in cnrN+1

## Idées de présentation
  - Print testsuite name on init ? on first test call ?
  - Print only failures in test suite by default
  - Remove colors of stdout & stderr
  - Prefix each line of stdout and stderr with stdout> & stderr>
  - Smart diff focusing on what is different for text comparison assertions
  - Display.TestTitle()
  - Display.TestOutcome()
  - Display.Suite()
  - Display.Global()
  - Display.ReportAll()
  - Display.ReportSuite()
  - Display.AssertionResult()
  - Display.Stdout()
  - Display.Stderr()



## Idées pour silent vs quiet ?!
  - @debug=LEVEL for debugging => which logger ?
  - replace @silent by @quiet
  - @quiet should produce minimal output (only @reports and @keepOutputs should output)
  - by default not inited suite is @verbose=1
  - by default inited suite is @verbose=0
  - verbose=0 => Print Failures & Errors Tests only with short assertion report
  - verbose=1 => Same + descriptive assertion report (cleaned & shorten stdout & stderr for failures & errors)
  - verbose=2 => Same + Print success titles
  - verbose=3 => Same + (cleaned & shorten stdout & stderr for success)

## Idées pour la gestion du Context
  - Context.PersistGlobal()
  - Context.PersistSuite()
  - Context.IncrementTestCount(outcome)
  - Context.UpdateContainerId(ctId, scope)
  - Functionnal options pattern : https://dev.to/kittipat1413/understanding-the-options-pattern-in-go-390c + persist a collection of options
  - Optional instead of pointer ? => cleaner for Merging

## Container abstraction ideas
- decorelate binary resolution => can store ans reuse it
- API ideas :
  - engine := containerz.Engine()
  - engine.Container(name).Run(image, cmdAndArgs).Detach().Rm() => containerz.Runner
  - engine.Container(name).Exec(cmdAndArgs).User(user) => containerz.Executer
  - engine.Container(name).Start().Timeout() => containerz.Starter
  - engine.Container(name).Stop() => containerz.Stopper
  - engine.Container(name).Rm().Force() => containerz.Rmer
  - engine.Container(name).Ps() => containerz.Pser
  - engine.Container(name).Inspect() => containerz.Insecter
  - engine.Build(dir).NoCache() => containerz.Builder
  - engine.Pull(image) => containerz.Puller
  - engine.Push(image) => containerz.Pusher

*/
