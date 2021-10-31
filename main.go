package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/innatical/pax/v2/util"
	"golang.org/x/sys/unix"
)

func main() {
	name, err := ioutil.TempDir("/tmp", "pax-chroot")
	if err != nil {
		panic(err)
	}

	if err := SetupChroot(name); err != nil {
		panic(err)
	}

	exit, err := OpenChroot(name)
	if err != nil {
		panic(err)
	}

	util.Install(name, "linux", "5.13", true)

	if err := exit(); err != nil {
		panic(err)
	}

	// if err := CleanupChroot(name); err != nil {
	// 	panic(err)
	// }
}

func OpenChroot(name string) (func() error, error) {
	root, err := os.Open("/")
	if err != nil {
		return nil, err
	}

	if err := os.Chdir(name); err != nil {
		return nil, err
	}

	if err := syscall.Chroot(name); err != nil {
		root.Close()
		return nil, err
	}

	exit := func() error {
		defer root.Close()
		if err := root.Chdir(); err != nil {
			return err
		}
		return syscall.Chroot(".")
	}

	return exit, nil

}

func SetupChroot(name string) error {
	procdir := filepath.Join(name, "proc")
	if err := os.Mkdir(procdir, 0777); err != nil {
		return err
	}
	if err := unix.Mount("/proc", procdir, "proc", 0, "rw"); err != nil {
		return err
	}

	sysdir := filepath.Join(name, "sys")
	if err := os.Mkdir(sysdir, 0777); err != nil {
		return err
	}
	if err := unix.Mount("/sys", sysdir, "sysfs", 0, "rw"); err != nil {
		return err
	}

	devdir := filepath.Join(name, "dev")
	if err := os.Mkdir(devdir, 0777); err != nil {
		return err
	}
	if err := unix.Mount("/dev", devdir, "none", unix.MS_BIND, ""); err != nil {
		return err
	}

	return nil
}

func CleanupChroot(name string) error {
	procdir := filepath.Join(name, "proc")
	if err := unix.Unmount(procdir, 0); err != nil {
		return err
	}

	sysdir := filepath.Join(name, "sys")
	if err := unix.Unmount(sysdir, 0); err != nil {
		return err
	}

	devdir := filepath.Join(name, "dev")
	if err := unix.Unmount(devdir, 0); err != nil {
		return err
	}

	return nil
}
