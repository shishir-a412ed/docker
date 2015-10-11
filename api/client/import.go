package client

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	Cli "github.com/docker/docker/cli"
	"github.com/docker/docker/opts"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/pkg/parsers"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/docker/docker/registry"
)

// CmdImport creates an empty filesystem image, imports the contents of the tarball into the image, and optionally tags the image.
//
// The URL argument is the address of a tarball (.tar, .tar.gz, .tgz, .bzip, .tar.xz, .txz) file or a path to local file relative to docker client. If the URL is '-', then the tar file is read from STDIN.
//
// Usage: docker import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]
// Usage for diff or metadata option: docker import [OPTIONS] - [CONTAINER ID|NAME]

func (cli *DockerCli) CmdImport(args ...string) error {
	cmd := Cli.Subcmd("import", []string{"file|URL|- [REPOSITORY[:TAG]]"}, Cli.DockerCommands["import"].Description, true)
	flChanges := opts.NewListOpts(nil)
	cmd.Var(&flChanges, []string{"c", "-change"}, "Apply Dockerfile instruction to the created image")
	message := cmd.String([]string{"m", "-message"}, "", "Set commit message for imported image")
	flDiff := cmd.String([]string{"-diff"}, "", "Import the diff of a container from a tarball")
	flMetadata := cmd.Bool([]string{"-metadata"}, false, "Import the metadata of a container from a tarball")
	cmd.Require(flag.Min, 1)

	cmd.ParseFlags(args, true)

	var (
		v          = url.Values{}
		src        = cmd.Arg(0)
		repository = cmd.Arg(1)
	)

	if *flMetadata && (*flDiff != "") {
		return errors.New("diff and metadata are sequential import options. Use diff after container's metadata is loaded")
	}

	if *flDiff != "" {
		v.Set("container", *flDiff)
	}

	v.Set("fromSrc", src)
	v.Set("repo", repository)
	v.Set("message", *message)

	for _, change := range flChanges.GetAll() {
		v.Add("changes", change)
	}
	if cmd.NArg() == 3 {
		fmt.Fprintf(cli.err, "[DEPRECATED] The format 'file|URL|- [REPOSITORY [TAG]]' has been deprecated. Please use file|URL|- [REPOSITORY[:TAG]]\n")
		v.Set("tag", cmd.Arg(2))
	}

	if repository != "" {
		//Check if the given image name can be resolved
		repo, _ := parsers.ParseRepositoryTag(repository)
		if err := registry.ValidateRepositoryName(repo); err != nil {
			return err
		}
	}

	var in io.Reader

	if src == "-" {
		in = cli.in
	} else if !urlutil.IsURL(src) {
		v.Set("fromSrc", "-")
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		defer file.Close()
		in = file

	}

	sopts := &streamOpts{
		rawTerminal: true,
		in:          in,
		out:         cli.out,
	}

	if *flDiff != "" {
		if _, err := cli.stream("POST", "/containers/diff?"+v.Encode(), sopts); err != nil {
			return err
		}
	} else if *flMetadata {
		if _, err := cli.stream("POST", "/containers/metadata", sopts); err != nil {
			return err
		}
	} else {
		_, err := cli.stream("POST", "/images/create?"+v.Encode(), sopts)
		return err
	}
	return nil
}
