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
	"os"
	"path/filepath"

	"mby.fr/utils/printz"
)

/**
Tester des commandes.

## cmdt @init
Initialize the test suite with the supplied configuration.
Options:
- @stopOnFailure : Stop test suite on first failure and report.
- @logOnSuccess : Always log cmd output.

This should initialize a uniq context shared by all test suite. How ?
- Export vars ?
- Produce a hidden file ?
- Produce a hidden dir ?

This could be ommited => testSuite name will then be empty and all following tests placed in this test suite.


## cmdt @report
Display a test suite report and return a failure if at least one test failed.

## cmdt <testName>
Launch a command test and do some assertion.
Assertions:
- @fail : cmd should fail (rc > 0)
- @success : cmd should succeed (rc = 0)
- @stdout=string : cmd stdout should be exactly message
- @stdout~string : cmd stdout should match message
- @stderr=string : cmd stderr should be exactly message
- @stderr~string : cmd stderr should match message
- @exists=path,perms,owners : a file should be produced at path with perms and owners

## Principes
- eval $( cmdt [testSuite] @init @stopOnFailure @logOnSuccess )
- cmdt <[testSuite/]testName> <myCommand> myArg1 ... myArgN @fail @rc=
- cmdt <[testSuite/]testName> <myCommand> myArg1 ... myArgN @success @stdout="MyOut" @stderr="MyErr" @exists="MyFile,Perms,Owners"
- cmdt [testSuite] @report

*/

/*
Done:
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


TODO:
Bugs:
- @FILEPATH empeche de tester un contenu qui commence par @ sans passer par un fichier. Il faudrait pouvoir échapper @ ou bien utiliser un operator dédié à la lécture d'un fichier (ex: @stderr@=FILEPATH @stdin@:FILEPATH)

Features :
- @mock stdin=@FILEPATH stdin:PARTIAL_CONTENT stdout=@FILEPATH @stderr=@FILEPATH
- @called[=:]CMD ARG_S,stdin=IN,count=N assertion => verify a mock was called
- silent ? quiet ? verbose ? an option to quiet errors as well ?
- use rule definitions in usage
- move seq into utils module
- order reports : report failures at the end of report
- rework failure description : hard to read (remove colors ? remove \n ?)
- improve printer. Writer stdout & stder with prefix stdout> stderr> in descriptions
- change default test suite with @init=foo => foo become default test suite
- Run en option les tests dans un conteneur => plus facile de mocké, meilleur reproductibilité des tests
	- Fournir une image pour tester l'éxecution dans cette image
	- Reset l'image à chaque test suite ou à chaque test en option
	- Mock les executable avec un chemin absolu
	- Utiliser une image par défaut pour les cas d'usage simples
- possibilité de passer un scénario ligne à ligne dans le stdin de cmdtest
	- cmdt cmd arg1 argN @scenario=filepath
	- pour chaque ligne du scenario concat la ligne du scenario avec les arguments fournit en paramétre de cmdt

- @runCount=N + @parallel=FORK_COUNT (min, max, median exec time) run in separate context or in same context (before/after) ?

- @fork=5 global config only by default instead of @parallel. Fork = 5 increment and decrement a seq file
- Clean old temp dir (older than 2 min ?)


- may chroot be interesting for tests ?
- mock web spawning a web server ?
- test port opening if daemon ; test sending data on port ???

## Idées pour la commande de mock
    insérer dans le $PATH un repertoire temp binMocked
    Pour chaque commande mockée on place un lien symbolique de la commande mocké vers le mock dans le repertoire binMocked
    il nous faut un programme mock qui consomme tous les args, compte les appels avec les args et répond de la manière dont il est configuré
    mock config :
        Exact call @mock="CMD,ARG_1,ARG_2,ARG_N;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in order not more
        Partial call @mock="CMD,ARG_1,ARG_2,ARG_N,*;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in order then more args
        Contains call @mock:"CMD,ARG_1,ARG_2,ARG_N;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in various order not more
        Default exit code @mock="CMD,*;exit=1"

## Idée pour executer les tests dans un conteneur
  - Une image par défaut busybox like (avec cmdt déjà à l'interieur ? pas forcément nécéssaire sauf pour le speed)
  - En général, il faut monter le binaire cmdt dans le conteneur et optionnelement l'ajouter au PATH
  - cmdt => run cmdt avec exactement les memes args dans un conteneur jetable démarré à l'instant par cmdt
  - avec quel owner démarrer cmdt dans le conteneur => default / option ??
  - Mock une commande absolue possible, mais nécéssite de déplacer la commande original pour la remplacer par un wrapper
  -


## Idées pour silent vs quiet ?!

*/

var stdPrinter printz.Printer

//var logger = logz.Default("cmdtest", 0)

func usage() {
	cmd := filepath.Base(os.Args[0])
	stdPrinter.Errf("cmdtest tool is usefull to test various scripts cli and command behaviors.\n")
	stdPrinter.Errf("You must initialize a test suite (%[1]s @init) before running tests and then report the test (%[1]s @report).\n", cmd)
	stdPrinter.Errf("usage: \t%s @init[=TEST_SUITE_NAME] [@CONFIG_1] ... [@CONFIG_N] \n", cmd)
	stdPrinter.Errf("usage: \t%s <COMMAND> [ARG_1] ... [ARG_N] [@CONFIG_1] ... [@CONFIG_N] [@ASSERTION_1] ... [@ASSERTION_N]\n", cmd)
	stdPrinter.Errf("usage: \t%s @report[=TEST_SUITE_NAME] \n", cmd)
	stdPrinter.Errf("\tCONFIG available: @ignore @stopOnFailure @keepStdout @keepStderr @keepOutputs @timeout=Duration @fork=N\n")
	stdPrinter.Errf("\tCOMMAND and ARGs: the command on which to run tests\n")
	stdPrinter.Errf("\tASSERTIONs available: @fail @success @exit=N @stdout= @stdout~ @stderr= @stderr~ @cmd= @exists=\n")
	stdPrinter.Errf("In complex cases assertions must be correlated by a token. You can generate a token with @init @printToken or @init @exportToken and supply it with @token=\n")
}

func main() {
	ProcessArgs(os.Args)
}
