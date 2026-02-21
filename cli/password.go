package cli

import (
	"errors"
	"fmt"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)


// ErrCanceled はユーザーが入力をキャンセルしたことを表します。
var ErrCanceled = errors.New("password input canceled")

// passwordModel は「パスワード入力だけ」を行うBubble Teaのモデルです。
type passwordModel struct {
	ti       textinput.Model
	done     bool
	canceled bool
	value    string
	err      error
}

func newPasswordModel() passwordModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 0
	
	// パスワード入力（伏字）
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•' // 好みで '*' などに
	
	return passwordModel{
		ti:     ti,
	}
}

func (m passwordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.value = m.ti.Value()
			m.done = true
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			m.canceled = true
			m.done = true
			return m, tea.Quit
		}
	}
	
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m passwordModel) View() string {
	if m.done { return "" }
	return fmt.Sprintf("Password %s\n(Enter: OK, Esc/Ctrl+C: Cancel)", m.ti.View())
}

// InputPassword は Bubble Tea を起動してパスワードを入力させ、確定した文字列を返します。
// キャンセル時は ErrCanceled を返します。
func InputPassword() (string, error) {
	m := newPasswordModel()
	
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil { return "", err }
	
	pm := finalModel.(passwordModel)
	if pm.canceled { return "", ErrCanceled }
	return pm.value, nil
}
