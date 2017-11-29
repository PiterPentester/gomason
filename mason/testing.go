package mason

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

// GovendorInstall  Installs govendor into the gopath indicated.
func GovendorInstall(gopath string, verbose bool) (err error) {
	if verbose {
		log.Printf("Installing Govendor with GOPATH=%s\n", gopath)
	}

	cmd := exec.Command("go", "get", "github.com/kardianos/govendor")

	runenv := append(os.Environ(), fmt.Sprintf("GOPATH=%s", gopath))

	cmd.Env = runenv

	err = cmd.Run()

	if err == nil {
		if verbose {
			log.Printf("Govendor installation complete\n\n")
		}
	}

	return err
}

// GovendorSync  Runs govendor sync in the diretory of your checked out code
func GovendorSync(gopath string, gomodule string, verbose bool) (err error) {
	wd := fmt.Sprintf("%s/src/%s", gopath, gomodule)

	if verbose {
		log.Printf("Changing working directory to: %s", wd)
	}

	err = os.Chdir(wd)

	if err != nil {
		log.Printf("Error changing working dir to %q: %s", wd, err)
		return err
	}

	govendor := fmt.Sprintf("%s/bin/govendor", gopath)

	cmd := exec.Command(govendor, "sync")

	runenv := append(os.Environ(), fmt.Sprintf("GOPATH=%s", gopath))

	cmd.Env = runenv

	if verbose {
		cwd, err := os.Getwd()
		if err != nil {
			err = errors.Wrap(err, "failed to get working directory")
			return err
		}

		log.Printf("Working directory: %s", cwd)
		log.Printf("GOPATH: %s", gopath)
		log.Printf("Running %s sync", govendor)
	}

	err = cmd.Run()

	if err == nil {
		if verbose {
			log.Printf("Govendor sync complete.\n\n")
		}
	}

	return err
}

// GoTest  Runs 'go test -v ./...' in the checked out code directory
func GoTest(gopath string, gomodule string, verbose bool) (err error) {
	wd := fmt.Sprintf("%s/src/%s", gopath, gomodule)

	if verbose {
		log.Printf("Changing working directory to %s.\n", wd)
	}

	err = os.Chdir(wd)

	if err != nil {
		log.Printf("Error changing working dir to %q: %s", wd, err)
		return err
	}

	if verbose {
		log.Printf("Running 'go test -v ./...'.\n\n")
	}

	cmd := exec.Command("go", "test", "-v", "./...")

	runenv := append(os.Environ(), fmt.Sprintf("GOPATH=%s", gopath))

	cmd.Env = runenv

	output, err := cmd.CombinedOutput()

	log.Printf(string(output))

	if verbose {
		log.Printf("Done with go test.\n\n")
	}

	return err
}

// WholeShebang Creates an ephemeral GOPATH, installs Govendor into it, checks out your code, and runs the tests.  The whole shebang, hence the name.
// Optionally, it will build and publish your code too while it has the whole GOPATH setup.
// Specify workdir if you want to speed things up (govendor sync can take a while), but it's up to you to keep it clean.
// If workDir is the empty string, it will use a temp file
func WholeShebang(workDir string, branch string, build bool, sign bool, publish bool, verbose bool) (buildmetadata Metadata, err error) {
	var actualWorkDir string

	cwd, err := os.Getwd()
	if err != nil {
		err = errors.Wrap(err, "Failed to get current working directory.")
		return buildmetadata, err
	}

	if workDir == "" {
		actualWorkDir, err = ioutil.TempDir("", "gomason")
		if err != nil {
			err = errors.Wrap(err, "Failed to create temp dir")
		}

		if verbose {
			log.Printf("Created temp dir %s", workDir)
		}

		defer os.RemoveAll(actualWorkDir)
	} else {
		actualWorkDir = workDir
	}

	buildmetadata.WorkDir = actualWorkDir

	gopath, err := CreateGoPath(actualWorkDir)
	if err != nil {
		return buildmetadata, err
	}

	buildmetadata.Path = gopath

	err = GovendorInstall(gopath, verbose)
	if err != nil {
		return buildmetadata, err
	}

	commandmetadata, err := ReadMetadata("metadata.json")
	if err != nil {
		err = errors.Wrap(err, "couldn't read package information from metadata.json.")
		return buildmetadata, err
	}

	buildmetadata.Package = commandmetadata.Package
	buildmetadata.Version = commandmetadata.Version

	giturl := GitSSHUrlFromPackage(commandmetadata.Package)

	buildmetadata.GitPath = giturl

	err = Checkout(gopath, commandmetadata.Package, branch, verbose)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("failed to checkout package %s at branch %s: %s", commandmetadata.Package, branch, err))
		return buildmetadata, err
	}

	err = GovendorSync(gopath, commandmetadata.Package, verbose)
	if err != nil {
		err = errors.Wrap(err, "error running govendor sync")
		return buildmetadata, err
	}

	err = GoTest(gopath, commandmetadata.Package, verbose)
	if err != nil {
		err = errors.Wrap(err, "error running go test")
		return buildmetadata, err
	}

	log.Printf("Success!\n\n")

	if build {
		Build(gopath, commandmetadata.Package, branch, verbose)
		parts := strings.Split(commandmetadata.Package, "/")

		binaryPrefix := parts[len(parts)-1]

		for _, arch := range commandmetadata.BuildTargets {
			archparts := strings.Split(arch, "/")

			osname := archparts[0]
			archname := archparts[1]

			workdir := fmt.Sprintf("%s/src/%s", gopath, commandmetadata.Package)
			binary := fmt.Sprintf("%s/%s_%s_%s", workdir, binaryPrefix, osname, archname)

			if _, err := os.Stat(binary); os.IsNotExist(err) {
				err = errors.New(fmt.Sprintf("Gox failed to build binary: %s\n", binary))
				return buildmetadata, err
			}

			destinationPath := fmt.Sprintf("%s/%s_%s_%s", cwd, binaryPrefix, osname, archname)

			err = os.Rename(binary, destinationPath)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("Failed to collect binary %s\n", binary))
				return buildmetadata, err
			}
		}
	}

	if sign {
		log.Printf("Signing not yet implemented.  Stay tuned\n")
		// sign binaries

	}

	if publish {
		log.Printf("Publish not yet implemented.  Stay tuned\n")
		// upload binaries
	}

	return buildmetadata, err
}
