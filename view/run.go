package view

import (
	"bakashier/cli"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func asViewModel(value tea.Model) (model, error) {
	result, ok := value.(model)
	if !ok {
		return model{}, fmt.Errorf("unexpected view model type %T", value)
	}
	return result, nil
}

func Run(mode cli.ModeType, receiveQueue <-chan MessageToView, sendQueue chan<- MessageToManager) (model, error) {
	m := model{
		mode:         mode,
		stop:         false,
		quit:         false,
		workers:      make(map[uint]workerStatus),
		receiveQueue: receiveQueue,
		sendQueue:    sendQueue,
	}
	program := tea.NewProgram(m)
	rm, err := program.Run()
	if err != nil {
		return m, err
	}
	return asViewModel(rm)
}
