package urlshortener

import "embed"

//go:embed db/migrations
var MigrationsFS embed.FS

//go:embed all:web/dist
var DistFS embed.FS

//go:embed static
var StaticFS embed.FS