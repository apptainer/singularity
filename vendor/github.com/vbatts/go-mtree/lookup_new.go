// +build go1.7

package mtree

import (
	"os/user"
)

var lookupGroupID = user.LookupGroupId
