package gomason

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
func Build(gopath string, meta Metadata, branch string, verbose bool) (err error) {
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

	if _, err := os.Stat(fmt.Sprintf("%s/%s/metadata.json", gopath, meta.Package)); os.IsNotExist(err) {
		err = Checkout(gopath, meta, branch, verbose)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Failed to checkout module: %s branch: %s ", meta.Package, branch))
			return err
		}
	}

	wd := fmt.Sprintf("%s/src/%s", gopath, meta.Package)

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
		log.Printf("Gox is: %s", gox)
	}

	metadatapath := fmt.Sprintf("%s/src/%s/metadata.json", gopath, meta.Package)

	md, err := ReadMetadata(metadatapath)
	if err != nil {
		err = errors.Wrap(err, "Failed to read metadata.json from checked out code")
		return err
	}

	for _, target := range md.BuildInfo.Targets {
		if verbose {
			log.Printf("Building target: %q\n", target.Name)
		}

		// This gets weird because go's exec shell doesn't like the arg format that gox expects
		// Building it thusly keeps the various quoting levels straight

		gopathenv := fmt.Sprintf("GOPATH=%s", gopath)
		runenv := append(os.Environ(), gopathenv)

		cgo := ""
		// build with cgo if we're told to do so.
		if target.Cgo {
			cgo = " -cgo"
		}

		for k, v := range target.Flags {
			runenv = append(runenv, fmt.Sprintf("%s=%s", k, v))
			if verbose {
				log.Printf("Build Flag: %s=%s", k, v)
			}
		}

		args := gox + cgo + ` -osarch="` + target.Name + `"` + " ./..."

		// Calling it through sh makes everything happy
		cmd := exec.Command("sh", "-c", args)

		cmd.Env = runenv

		if verbose {
			log.Printf("Running gox with: %s", args)
		}

		out, err := cmd.CombinedOutput()

		log.Printf("%s\n", string(out))

		if err != nil {
			log.Printf("Build error: %s\n", err.Error())
			return err
		}

		if verbose {
			log.Printf("Gox build complete and successful.\n\n")
		}

	}

	err = BuildExtras(md, wd, verbose)
	if err != nil {
		err = errors.Wrapf(err, "Failed to build extras")
		return err

	}

	return err
}

// BuildExtras builds the extra artifacts specified in the metadata.json
func BuildExtras(meta Metadata, workdir string, verbose bool) (err error) {
	if verbose {
		log.Printf("Building Extra Artifacts")
	}

	for _, extra := range meta.BuildInfo.Extras {
		templateName := fmt.Sprintf("%s/%s", workdir, extra.Template)
		outputFileName := fmt.Sprintf("%s/%s", workdir, extra.FileName)
		executable := extra.Executable

		if verbose {
			fmt.Printf("Reading template from %s\n", templateName)
			fmt.Printf("Writing to %s\n", outputFileName)
		}

		var mode os.FileMode

		if executable {
			mode = 0755
		} else {
			mode = 0644
		}

		tmplBytes, err := ioutil.ReadFile(templateName)
		if err != nil {
			err = errors.Wrapf(err, "failed to read template file %s", templateName)
			return err
		}

		output, err := ParseTemplateForMetadata(string(tmplBytes), meta)
		if err != nil {
			err = errors.Wrapf(err, "failed to inject metadata into template text")
			return err
		}

		err = ioutil.WriteFile(outputFileName, []byte(output), mode)
		if err != nil {
			err = errors.Wrapf(err, "failed to write file %s", outputFileName)
			return err
		}
	}

	return err
}
