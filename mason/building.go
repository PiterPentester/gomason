package mason

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
	"strings"
)

// GoxInstall Installs github.com/mitchellh/gox, the go cross compiler
func GoxInstall(gopath string, verbose bool) (err error) {
	if verbose {
		log.Printf("Installing gox with GOPATH=%s\n", gopath)
	}

	gocommand, err := exec.LookPath("go")
	if err != nil {
		err = errors.Wrap(err, "Failed to find go binary")
		return err
	}

	cmd := exec.Command(gocommand, "get", "github.com/mitchellh/gox")

	env := append(os.Environ(), fmt.Sprintf("GOPATH=%s", gopath))

	cmd.Env = env

	err = cmd.Run()
	if err == nil {
		if verbose {
			log.Printf("Gox successfully installed.\n\n")
		}
	}

	return err
}

// Build  Builds the package.  Builds binaries for the architectures listed in the metadata.json file
func Build(gopath string, gomodule string, branch string, verbose bool) (err error) {
	if verbose {
		log.Printf("Checking to see that gox is installed.\n")
	}
	// Install gox if it's not already there
	if _, err := os.Stat(fmt.Sprintf("%s/go/bin/gox", gopath)); os.IsNotExist(err) {
		err = GoxInstall(gopath, verbose)
		if err != nil {
			err = errors.Wrap(err, "Failed to install gox")
			return err
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/%s/metadata.json", gopath, gomodule)); os.IsNotExist(err) {
		err = Checkout(gopath, gomodule, branch, verbose)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Failed to checkout module: %s branch: %s ", gomodule, branch))
			return err
		}
	}

	wd := fmt.Sprintf("%s/src/%s", gopath, gomodule)

	if verbose {
		log.Printf("Changing working directory to: %s", wd)
	}

	err = os.Chdir(wd)

	if err != nil {
		log.Printf("Error changing working dir to %q: %s", wd, err)
		return err
	}

	gox := fmt.Sprintf("%s/bin/gox", gopath)

	if verbose {
		fmt.Printf("Gox is: %s", gox)
	}

	metadatapath := fmt.Sprintf("%s/src/%s/metadata.json", gopath, gomodule)

	md, err := ReadMetadata(metadatapath)
	if err != nil {
		err = errors.Wrap(err, "Failed to read metadata.json from checked out code")
		return err
	}

	targets := md.BuildTargets

	targetstring := strings.Join(targets, " ")

	// This gets weird because go's exec shell doesn't like the arg format that gox expects
	// Building it thusly keeps the various quoting levels straight

	args := gox + ` -osarch="` + targetstring + `"`

	// Calling it through bash makes everything happy
	cmd := exec.Command("bash", "-c", args)

	gopathenv := fmt.Sprintf("GOPATH=%s", gopath)

	runenv := append(os.Environ(), gopathenv)

	cmd.Env = runenv

	if verbose {
		log.Printf("Running gox....\n")
	}

	out, err := cmd.CombinedOutput()

	log.Printf("%s\n", string(out))

	if err == nil {
		if verbose {
			log.Printf("Gox build complete and successful.\n\n")
		}
	}

	return err
}
