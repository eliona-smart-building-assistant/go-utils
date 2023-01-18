# Go Utils #

Go utils is a part of the Eliona App SDK. It provides a Go library to make it easier to work with the Go language.

The library contains handy functions to access database resources, Kafka topics and HTTP endpoints.
Besides, the library offers useful tools like logging and process control.

## Installation ##

To get go-utils you can use command line:

```bash
go get github.com/eliona-smart-building-assistant/go-utils
```

or you define import in go files:

```go
import "github.com/eliona-smart-building-assistant/go-utils"
```

and run `go get` without parameters.

## Usage ##
 
- [Logging](log) for logging purposes
- [Database](db) to access databases
- [Http](http) to handle web requests
- [Kafka](kafka) to handle kafka topics
