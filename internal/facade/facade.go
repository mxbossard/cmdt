package facade

type Facade interface {
	ManageError(err error)
	GlobalAction() error
	SuiteAction() error
	TestAction() error
	ReportAction() error
}

/*
## Should do the right thing in respect to the context :
- scope (global, suite, test)
- display ?
- sync / async (some port on client side, other parts on daemon side)
- container or not
- wait or not ?
- timeout ?

*/

/*
## Cli only process
- Validate args
- Build ActionDef
=> implemented in Facade ?

## Common process (cli & daemon)
- Process ActionDef (global?, suite?, test, report?)
- Display
=> implemented in Service ?

*/
