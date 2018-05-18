/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package fs

import "testing"

func TestIsFile(t *testing.T) {
	if IsFile("/etc/passwd") != true {
		t.Errorf("IsFile returns false for file")
	}
}

func TestIsDir(t *testing.T) {
	if IsDir("/etc") != true {
		t.Errorf("IsDir returns false for directory")
	}
}

func TestIsLink(t *testing.T) {
	if IsLink("/proc/mounts") != true {
		t.Errorf("IsLink returns false for link")
	}
}

func TestIsOwner(t *testing.T) {
	if IsOwner("/etc/passwd", 0) != true {
		t.Errorf("IsOwner returns false for /etc/passwd owner")
	}
}

func TestIsExec(t *testing.T) {
	if IsExec("/bin/ls") != true {
		t.Errorf("IsExec returns false for /bin/ls execution bit")
	}
}

func TestIsSuid(t *testing.T) {
	if IsSuid("/bin/su") != true {
		t.Errorf("IsSuid returns false for /bin/su setuid bit")
	}
}
