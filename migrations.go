package main

import "embed"

//go:embed db/migrations
var migrationsFS embed.FS