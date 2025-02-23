package help

/*
  ## Generic Help for basic usage if no args supplied ex: cmdt @help
  - display cmdt short description + documentation (several paragraphs)
  - list actions available with short description
  - list global config with short description
  - list global examples, workflow examples

  ## Help by action ; ex: cmdt @help init / cmdt @help @init / cmdt @help --init
  - display action short description + documentation
  - list action configs whith short description
  - list action examples

  ## Help by global config ; ex: cmdt @help @async
  - display config short description + documentation
  - list examples

  ## Help by scoped config ; ex: cmdt @help test timeout
  - display config short description + documentation
  - list scopes available for that config
  - list examples

  ## Need some tree/graph
  Generate actions list, scoped configs, config available scopes from tree

  cmdt
    - action=test
	  - assert=fail
	    - scopes: [test]
	- action=suite
	- config=async
	  - scopes: [global, suite]
	- config=suiteTimeout
	  - scopes: [global, suite]
	- config=timeout
	  - scopes: [global, suite, test]


*/

type documentation struct {
	kind          string // action, config, assert
	description   string
	documentation []string
	examples      []string
	//scopes        []string
}

// Produce a help message adapted to supplied args
func Usage(args ...string) string {
	panic("not implemented yet")
}

func childs(args ...string) documentation {
	panic("not implemented yet")
}
