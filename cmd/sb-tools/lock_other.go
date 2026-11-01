//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd

package main

import (
	"errors"
	"os"
)

func backupLock(p string) (func(), error) {
	p += ".directory"
	if e := os.Mkdir(p, 0700); e != nil {
		return nil, errors.New("backup lock unavailable; inspect abandoned lock manually")
	}
	return func() { os.Remove(p) }, nil
}
