package view

import "testing"

func TestModelBoundsStoredErrorLogs(t *testing.T) {
	const maxStoredErrorLogs = 1000
	m := model{workers: make(map[uint]workerStatus)}
	for i := 0; i < maxStoredErrorLogs+100; i++ {
		updated, _ := m.Update(MessageToView{MsgType: ERROR, Detail: "error"})
		m = updated.(model)
	}

	if len(m.ErrorLog) > maxStoredErrorLogs+1 {
		t.Fatalf("stored error logs = %d, want at most %d plus omission marker", len(m.ErrorLog), maxStoredErrorLogs)
	}
}

func TestAsViewModelRejectsUnexpectedType(t *testing.T) {
	if _, err := asViewModel(nil); err == nil {
		t.Fatal("expected unexpected model type error")
	}
}
