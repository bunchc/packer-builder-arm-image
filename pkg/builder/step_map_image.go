package builder

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

type stepMapImage struct {
	ImageKey  string
	ResultKey string
}

func (s *stepMapImage) Run(state multistep.StateBag) multistep.StepAction {
	// Read our value and assert that it is they type we want
	image := state.Get(s.ImageKey).(string)
	ui := state.Get("ui").(packer.Ui)

	ui.Message(fmt.Sprintf("mappping %s", image))
	if run(state, fmt.Sprintf(
		"kpartx -s -a %s",
		image)) != nil {
		return multistep.ActionHalt
	}

	out, err := exec.Command("kpartx", "-l", image).CombinedOutput()
	ui.Say(fmt.Sprintf("kpartx -l: %s", string(out)))
	if err != nil {
		ui.Error(fmt.Sprintf("error kaprts -l %v: %s", err, string(out)))
		s.Cleanup(state)
		return multistep.ActionHalt
	}

	// get the loopback device for the partitions
	// kpartx -l output looks like this:
	/*
		loop2p1 : 0 85045 /dev/loop2 8192
		loop2p2 : 0 3534848 /dev/loop2 94208
	*/
	lines := strings.Split(string(out), "\n")

	var partitions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		device := strings.Split(string(line), " : ")
		if len(device) != 2 {
			ui.Error("bad kpartx output: " + string(out))
			s.Cleanup(state)
			return multistep.ActionHalt
		}
		partitions = append(partitions, "/dev/mapper/"+device[0])
	}

	state.Put(s.ResultKey, partitions)

	return multistep.ActionContinue
}

func (s *stepMapImage) Cleanup(state multistep.StateBag) {
	image := state.Get(s.ImageKey).(string)
	run(state, fmt.Sprintf("kpartx -d %s", image))
}
