package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/innatical/pax/v2/util"
	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
)

var errorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))

func main() {
	app := &cli.App {
		Name:      "pax-chroot",
		Usage:     "Pax Chroot Utility",
		UsageText: "pax-chroot [options]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "command",
				Value: "bash",
				Usage: "The command to run after entering the chroot",
				Aliases: []string{"c"},
			},
			&cli.PathFlag{
				Name: "config",
				Value: "PAXCHROOT",
				Usage: "The config file to use when creating a chroot",
				Aliases: []string{"f"},
			},
			&cli.BoolFlag{
				Name: "mount-root",
				Value: false,
				Usage: "Mount the host's root to /mnt in the chroot",
				Aliases: []string{"r"},
			},
			&cli.BoolFlag{
				Name: "use-current-dir",
				Value: false,
				Usage: "Change the working directory in the chroot to the current dir, must be combined with --mount-root",
				Aliases: []string{"u"},
			},
		},
		Action: mainCommand,
	}

	if err := app.Run(os.Args); err != nil {
		println(errorStyle.Render("Error: ") + err.Error())
		os.Exit(1)
	}
}

func mainCommand(c *cli.Context) error {
	name, err := ioutil.TempDir("/tmp", "pax-chroot")
	if err != nil {
		return err
	}

	if err := SetupChroot(name); err != nil {
		return err
	}

	if c.Bool("mount-root") {
		if err := BindMount(name, "/mnt", "/"); err != nil {
			return nil
		}
	}

	err = Cp(filepath.Join(os.Getenv("HOME"), "/.apkg/paxsources.list"), filepath.Join(name, "paxsources.list"))
	if err != nil {
		return err
	}

	configFile := c.Path("config")
	config, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	for _, pkg := range strings.Split(string(config), "\n") {
		parsed := strings.Split(pkg, "@")

		if pkg == "" {
			continue
		}
		
		println("Installing " + parsed[0] + " in chroot...")
		if len(parsed) == 1 {
			if err := util.Install(name, parsed[0], "", true); err != nil {
				return nil
			}
		} else {
			if err := util.Install(name, parsed[0], parsed[1], true); err != nil {
				return nil
			}
		}
	}

	curr, err := os.Getwd()

	if err != nil {
		return nil
	}

	exit, err := OpenChroot(name)
	if err != nil {
		return err
	}

	if c.Bool("use-current-dir") {
		if err := os.Chdir(filepath.Join("/mnt", curr)); err != nil {
			return err
		}
	}

	cmd := exec.Command(c.String("command"))
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	if err := exit(); err != nil {
		return err
	}

	if err := CleanupChroot(name); err != nil {
		return err
	}

	if c.Bool("mount-root") {
		if err := UnmountBind(name, "/mnt"); err != nil {
			return nil
		}
	}

	return err
}

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