package main

import (
	"fmt"
	"time"

	quicklog "github.com/quicklog-io/quicklog-go"
)

var (
	Quicklog = quicklog.Quicklog
	TagTrace = quicklog.TagTrace
	TraceCtx = quicklog.TraceCtx
)

func main() {
	quicklog.Configure(quicklog.Config{
		ProjectID: 12345,
		ApiKey:    "my-api-key",
		Source:    "my-program"})

	traceCtx := TraceCtx("user:me", "", "")

	extra := make(map[string]interface{})
	extra["key"] = "value"
	tags := []string{"name1:value1", "value", "name1:value:with:colons", ":value:with:colons"}

	err := Quicklog(time.Now(), "a-type", "object:1", "target:2", extra, traceCtx, tags...)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("OK: Logged.\n")
	}
}
