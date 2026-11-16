//go:build !unix

package proc

import "os/exec"

// killGroupOnCancel keeps exec's default of killing the direct child: process
// groups are a Unix facility.
func killGroupOnCancel(*exec.Cmd) {}
