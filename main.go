package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
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

	err = cp(filepath.Join(os.Getenv("HOME"), "/.apkg/paxsources.list"), filepath.Join(name, "paxsources.list"))
	if err != nil {
		panic(err)
	}

	if err := util.Install(name, "pax", "2.0.3", false); err != nil {
		panic(err)
	}
	if err := util.Install(name, "apkg", "2.0.11", false); err != nil {
		panic(err)
	}
	if err := util.Install(name, "gcc", "11.2.0", false); err != nil {
		panic(err)
	}
	if err := util.Install(name, "glibc", "2.34.0", false); err != nil {
		panic(err)
	}
	if err := util.Install(name, "bash", "5.1.8", false); err != nil {
		panic(err)
	}
	if err := util.Install(name, "ncurses", "6.2.0", false); err != nil {
		panic(err)
	}
	if err := util.Install(name, "readline", "8.1.0", false); err != nil {
		panic(err)
	}

	exit, err := OpenChroot(name)
	if err != nil {
		panic(err)
	}

	bash := exec.Command("bash")
	bash.Stdout = os.Stdout
	bash.Stdin = os.Stdin
	bash.Stderr = os.Stderr
	_ = bash.Run()

	if err := exit(); err != nil {
		panic(err)
	}

	// if err := CleanupChroot(name); err != nil {
	// 	panic(err)
	// }
}

func cp(from string, to string) error {
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}

	toFile, err := os.Create(to)
	if err != nil {
		return err
	}
	defer toFile.Close()

	if _, err = io.Copy(toFile, fromFile); err != nil {
		return err
	}

	if err = toFile.Sync(); err != nil {
		return err
	}

	return nil
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
