package flags

import (
	"fmt"
	"strings"

	"github.com/freakmaxi/kertish-dos/basics/common"
	"github.com/freakmaxi/kertish-dos/basics/terminal"
	"github.com/freakmaxi/kertish-dos/fs-tool/dos"
)

type changeDirectoryCommand struct {
	headAddresses []string
	output        terminal.Output
	target        string

	CurrentFolder *common.Folder
}

func NewChangeDirectory(headAddresses []string, output terminal.Output, target string) Execution {
	return &changeDirectoryCommand{
		headAddresses: headAddresses,
		output:        output,
		target:        target,
	}
}

func (c *changeDirectoryCommand) Parse() error {
	if len(c.target) == 0 {
		return fmt.Errorf("cd command needs only target parameter")
	}

	return nil
}

func (c *changeDirectoryCommand) PrintUsage() {
	c.output.Println("  cd          Change folders.")
	c.output.Println("              Ex: cd [target]")
	c.output.Println("")
	c.output.Refresh()
}

func (c *changeDirectoryCommand) Name() string {
	return "cd"
}

func (c *changeDirectoryCommand) Execute() error {
	if strings.Index(c.target, local) == 0 {
		return fmt.Errorf("cd works only for dos folder(s)")
	}

	anim := common.NewAnimation(c.output, "processing...")
	anim.Start()

	folder, err := dos.List(c.headAddresses, c.target, false)
	if err != nil {
		anim.Cancel()
		return err
	}
	anim.Stop()

	c.CurrentFolder = folder

	return nil
}

var _ Execution = &changeDirectoryCommand{}
