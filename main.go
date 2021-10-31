package main

import (
	"io/ioutil"

	"golang.org/x/sys/unix"
)

func main() {

}

func setup_chroot() error {
	name, err := ioutil.TempDir("/tmp", "pax-chroot")
	if err != nil {
		return err
	}

	if err := unix.Mount("/proc", name+"/proc", "proc"); err != nil {
		return err
	}

	if err := unix.Mount("/sys", name+"/sys", "sysfs"); err != nil {
		return err
	}
}
