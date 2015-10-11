package client

import (
	"errors"
	"os"

	Cli "github.com/docker/docker/cli"
	flag "github.com/docker/docker/pkg/mflag"
)

// CmdExport exports a filesystem as a tar archive.
//
// The tar archive is streamed to STDOUT by default or written to a file.
//
// Usage: docker export [OPTIONS] CONTAINER
func (cli *DockerCli) CmdExport(args ...string) error {
	cmd := Cli.Subcmd("export", []string{"CONTAINER"}, Cli.DockerCommands["export"].Description, true)
	outfile := cmd.String([]string{"o", "-output"}, "", "Write to a file, instead of STDOUT")
	diff := cmd.Bool([]string{"-diff"}, false, "Archive the diff of a container into a tarball")
	metadata := cmd.Bool([]string{"-metadata"}, false, "Archive the metadata of a container into a tarball")
	cmd.Require(flag.Exact, 1)

	cmd.ParseFlags(args, true)

	image := cmd.Arg(0)

	var (
		output = cli.out
		err    error
	)

	if *diff && *metadata {
		return errors.New("diff and metadata are mutually exclusive options. Use either one of them, not both")
	}

	if *outfile != "" {
		output, err = os.Create(*outfile)
		if err != nil {
			return err
		}
	} else if cli.isTerminalOut {
		return errors.New("Cowardly refusing to save to a terminal. Use the -o flag or redirect.")
	}

	sopts := &streamOpts{
		rawTerminal: true,
		out:         output,
	}

	if *metadata {
		if _, err := cli.stream("GET", "/containers/"+image+"/metadata", sopts); err != nil {
			return err
		}
	} else if *diff {
		if _, err := cli.stream("GET", "/containers/"+image+"/diff", sopts); err != nil {
			return err
		}
	} else {
		if _, err := cli.stream("GET", "/containers/"+image+"/export", sopts); err != nil {
			return err
		}
	}

	return nil
}
