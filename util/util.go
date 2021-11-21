package util

import (
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func Cp(from string, to string) error {
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

	resolv := filepath.Join(name, "etc/resolv.conf")
	if err := os.Mkdir(filepath.Join(name, "etc"), 0777); err != nil {
		return err
	}
	if err := unix.Mount("/etc/resolv.conf", resolv, "none", unix.MS_BIND, ""); err != nil {
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

	resolv := filepath.Join(name, "etc/resolv.conf")
	if err := unix.Unmount(resolv, 0); err != nil {
		return err
	}

	return nil
}

func BindMount(root string, from string, to string) error {
	dir := filepath.Join(root, from)
	if err := os.Mkdir(dir, 0777); err != nil {
		return err
	}
	if err := unix.Mount(to, dir, "none", unix.MS_BIND, ""); err != nil {
		return err
	}

	return nil
}

func UnmountBind(root string, from string) error {
	dir := filepath.Join(root, from)
	if err := unix.Unmount(dir, 0); err != nil {
		return err
	}

	return nil
}
