package view

import (
	"fmt"
	"sort"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"bakashier/cli"
	"bakashier/constants"
)


type workerStatus struct {
	srcDirectory string
	distDirectory string
	srcFile string
	distFile string
}


type model struct {
	mode cli.ModeType
	stop bool
	quit bool
	workers map[uint]workerStatus  // 各ワーカーの状態
	errorLog []string              // エラーログ
	receiveQueue <-chan MessageToView
	sendQueue chan<- MessageToDispatcher
}

type channelClosedMsg struct {}

const MAX_ERROR_LOGS int = 4

func receiveMessageCmd(queue <-chan MessageToView) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-queue
		if !ok { return channelClosedMsg{} }
		return msg
	}
}

func (m model) Init() tea.Cmd {
	return receiveMessageCmd(m.receiveQueue)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MessageToView:
		// coreからのメッセージを処理する
		switch msg.MsgType {
		case ADD_WORKER:
			m.workers[msg.WorkerId] = workerStatus{}
		case START_DIR:
			if status, ok := m.workers[msg.WorkerId]; ok {
				status.srcDirectory = msg.SrcPath
				status.distDirectory = msg.DistPath
				status.srcFile = ""
				status.distFile = ""
				m.workers[msg.WorkerId] = status
			}
		case FINISH_DIR:
			if status, ok := m.workers[msg.WorkerId]; ok {
				status.srcDirectory = ""
				status.distDirectory = ""
				status.srcFile = ""
				status.distFile = ""
				m.workers[msg.WorkerId] = status
			}
		case START_FILE:
			if status, ok := m.workers[msg.WorkerId]; ok {
				status.srcFile = msg.SrcPath
				status.distFile = msg.DistPath
				m.workers[msg.WorkerId] = status
			}
		case FINISH_FILE:
			if status, ok := m.workers[msg.WorkerId]; ok {
				status.srcFile = ""
				status.distFile = ""
				m.workers[msg.WorkerId] = status
			}
		case ERROR:
			m.errorLog = append(m.errorLog, msg.Detail)
			if len(m.errorLog) > MAX_ERROR_LOGS {
				m.errorLog = m.errorLog[1:]
			}
		case FINISHED:
			return m, tea.Quit
		}
		return m, receiveMessageCmd(m.receiveQueue)
	case channelClosedMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.stop = true
			m.sendQueue <- MessageToDispatcher{MsgType: STOP_WORKERS}
		case "r":
			m.stop = false
			m.sendQueue <- MessageToDispatcher{MsgType: RESUME_WORKERS}
		case "q":
			m.quit = true
			m.sendQueue <- MessageToDispatcher{MsgType: TERMINATION}
		}
	}
	
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	var modeLabel string = "unknown"
	var working bool = false
	var workerIds []uint = make([]uint, 0, len(m.workers))
	
	switch m.mode {
	case cli.ModeBackup:
		modeLabel = "Backup"
	case cli.ModeRestore:
		modeLabel = "Restore"
	}
	
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	
	for workerId := range m.workers {
		workerIds = append(workerIds, workerId)
	}
	sort.Slice(workerIds, func(i, j int) bool {
		return workerIds[i] < workerIds[j]
	})
	
	b.WriteString(fmt.Sprintf("Total Workers %d\n", len(m.workers)))
	for _, workerId := range workerIds {
		workerStatus := m.workers[workerId]
		if workerStatus.srcFile != "" {
			working = true
			b.WriteString(fmt.Sprintf("Worker %2d: %c%s%c\n", workerId, '"', workerStatus.srcFile, '"'))
			b.WriteString(fmt.Sprintf("         -> %c%s%c\n", '"', workerStatus.distFile, '"'))
		} else if workerStatus.srcDirectory != "" {
			working = true
			b.WriteString(fmt.Sprintf("Worker %2d: %c%s%c\n", workerId, '"', workerStatus.srcDirectory, '"'))
			b.WriteString(fmt.Sprintf("         -> %c%s%c\n", '"', workerStatus.distDirectory, '"'))
		} else {
			b.WriteString(fmt.Sprintf("Worker %2d: %s\n\n", workerId, gray.Render("(idle)")))
		}
	}
	b.WriteString("--------------------\n")
	for i := 0; i < MAX_ERROR_LOGS; i++ {
		if i < len(m.errorLog) {
			b.WriteString(red.Render(m.errorLog[i]))
		}
		b.WriteString("\n")
	}
	b.WriteString("--------------------\n")
	if m.quit {
		if working {
			b.WriteString(gray.Render("quitting...") + " \n")
		} else {
			b.WriteString("\n")
		}
	} else if m.stop {
		if working {
			b.WriteString(gray.Render("stopping...") + " \n")
		} else {
			b.WriteString("(R) resume  (Q) quit\n")
		}
	} else {
		b.WriteString("(S) stop    (Q) quit\n")
	}
	b.WriteString(fmt.Sprintf("%s v%s\n", constants.APP_NAME, constants.APP_VERSION))
	
	if !working {
		if !m.stop && !m.quit {
			b.Reset()
			b.WriteString(fmt.Sprintf("%s finished", modeLabel))
			return b.String()
		}
	}
	return b.String()
}
