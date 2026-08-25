// SPDX-License-Identifier: AGPL-3.0-only

package nodeclient

import _ "embed"

//go:embed client.sh
var script []byte

func Script() []byte {
	result := make([]byte, len(script))
	copy(result, script)
	return result
}
