package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"

	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"runtime"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SequelAceLoc is where we expect to find the Sequel Ace.app
// It's global so it can be mocked in testing.
var SequelAceLoc = "/Applications/Sequel Ace.app"

// DdevSequelAceCmd represents the sequelpro command
var DdevSequelAceCmd = &cobra.Command{
	Use:     "sequelace",
	Short:   "Connect sequelace to a project database",
	Long:    `A helper command for using sequelace (macOS database browser) with a running DDEV-Local project's database'.`,
	Example: `ddev sequelace`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			output.UserOut.Fatalf("invalid arguments to sequelace command: %v", args)
		}

		out, err := handleSequelAceCommand(SequelAceLoc)
		if err != nil {
			output.UserOut.Fatalf("Could not run sequelace command: %s", err)
		}
		util.Success(out)
	},
}

// handleSequelAceCommand() is the "real" handler for the real command
func handleSequelAceCommand(appLocation string) (string, error) {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return "", err
	}

	if app.SiteStatus() != ddevapp.SiteRunning {
		return "", errors.New("project is not running. The project must be running to create a Sequel Ace connection")
	}

	db, err := app.FindContainerByType("db")
	if err != nil {
		return "", err
	}

	dbPrivatePort, err := strconv.ParseInt(ddevapp.GetPort("db"), 10, 64)
	if err != nil {
		return "", err
	}
	dbPublishPort := fmt.Sprint(dockerutil.GetPublishedPort(dbPrivatePort, *db))

	tmpFilePath := filepath.Join(app.GetAppRoot(), ".ddev/sequelace.spf")
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		output.UserOut.Fatalln(err)
	}
	defer util.CheckClose(tmpFile)

	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return "", err
	}
	_, err = tmpFile.WriteString(fmt.Sprintf(
		ddevapp.SequelproTemplate,
		"db",           //dbname
		dockerIP,       //host
		app.HostName(), //connection name
		"db",           // dbpass
		dbPublishPort,  // port
		"db",           //dbuser
	))
	util.CheckErr(err)

	err = exec.Command("open", tmpFilePath).Run()
	if err != nil {
		return "", err
	}
	return "sequelace command finished successfully!", nil
}

// dummyDevSequelAceCmd represents the "not available" sequelace command
var dummyDevSequelAceCmd = &cobra.Command{
	Use:   "sequelace",
	Short: "This command is not available since sequel ace.app is not installed",
	Long:  `Where installed, "ddev sequelace" launches the sequel ace database browser`,
	Run: func(cmd *cobra.Command, args []string) {
		util.Failed("The sequelace command is not available because sequel ace.app is not detected on your workstation")

	},
}

// init installs the real command if it's available, otherwise dummy command (if on OSX), otherwise no command
func init() {
	switch {
	case detectSequelAce():
		app, err := ddevapp.GetActiveApp("")
		if err == nil && app != nil && !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
			RootCmd.AddCommand(DdevSequelAceCmd)
		}
	case runtime.GOOS == "darwin":
		RootCmd.AddCommand(dummyDevSequelAceCmd)
	}
}

// detectSequelAce looks for the sequel ace app in /Applications; returns true if found
func detectSequelAce() bool {
	if _, err := os.Stat(SequelAceLoc); err == nil {
		return true
	}
	return false
}
