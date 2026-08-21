// SPDX-License-Identifier: AGPL-3.0-only

package webassets

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

func Dist() fs.FS {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}

	return dist
}
