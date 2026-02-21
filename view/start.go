package view

import (
	"bakashier/cli"
	
	tea "github.com/charmbracelet/bubbletea"
)


func NewProgram(mode cli.ModeType, receiveQueue <-chan MessageToView, sendQueue chan<- MessageToManager) *tea.Program {
	m := model{
		mode:         mode,
		stop:         false,
		quit:         false,
		workers:      make(map[uint]workerStatus),
		receiveQueue: receiveQueue,
		sendQueue:    sendQueue,
	}
	return tea.NewProgram(m)
}
