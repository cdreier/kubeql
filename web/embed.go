// Package web embeds the Vite production build for serving from the Go binary.
package web

import "embed"

// DistFS is the contents of web/dist (index.html + hashed assets).
// Build with: cd web && npm run build  (or make build-web)
//
//go:embed all:dist
var DistFS embed.FS
