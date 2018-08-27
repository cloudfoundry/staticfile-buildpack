// +build !windows

package libbuildpack_test

import "syscall"

func umask(newmask int) (oldmask int) {
	return syscall.Umask(newmask)
}
