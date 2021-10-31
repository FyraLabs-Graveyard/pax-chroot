package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/innatical/pax-chroot/util"
	pax "github.com/innatical/pax/v2/util"
	"github.com/urfave/cli/v2"
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

	if err := util.SetupChroot(name); err != nil {
		return err
	}

	if c.Bool("mount-root") {
		if err := util.BindMount(name, "/mnt", "/"); err != nil {
			return nil
		}
	}

	err = util.Cp(filepath.Join(os.Getenv("HOME"), "/.apkg/paxsources.list"), filepath.Join(name, "paxsources.list"))
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
			if err := pax.Install(name, parsed[0], "", true); err != nil {
				return nil
			}
		} else {
			if err := pax.Install(name, parsed[0], parsed[1], true); err != nil {
				return nil
			}
		}
	}

	curr, err := os.Getwd()

	if err != nil {
		return nil
	}

	exit, err := util.OpenChroot(name)
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

	if err := util.CleanupChroot(name); err != nil {
		return err
	}

	if c.Bool("mount-root") {
		if err := util.UnmountBind(name, "/mnt"); err != nil {
			return nil
		}
	}

	return err
}