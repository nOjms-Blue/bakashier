package view

import (
	"bakashier/cli"
	
	tea "github.com/charmbracelet/bubbletea"
)


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
	return rm.(model), err
}
