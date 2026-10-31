package main

// The offline import tool is the connector's only command.
import (
	_ "example.test/mod/internal/integration/legacy/snapshot"
	_ "example.test/mod/internal/integration/legacy/spec"
)
