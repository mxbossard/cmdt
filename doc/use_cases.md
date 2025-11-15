## Standard

### Unit Tests
#### Basic
- cmdt @global 
- cmdt @init=suite
- cmdt @test=foo
- cmdt @report

#### Reopen
- cmdt @global 
- cmdt @init=suite
- cmdt @test=foo
- cmdt @report
- cmdt @test=bar
- cmdt @report

#### Async
- cmdt @global 
- cmdt @init=suite @async
- cmdt @test=foo
- cmdt @test=bar
- cmdt @report