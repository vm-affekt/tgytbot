package app

import "sync"

type UserDialogState struct {
	mu             *sync.Mutex
	dialogByUserID map[int64]Dialog
}

func NewUserDialogState() *UserDialogState {
	return &UserDialogState{
		dialogByUserID: make(map[int64]Dialog),
		mu:             new(sync.Mutex),
	}
}

func (uds *UserDialogState) FindDialogByUser(userID int64) Dialog {
	uds.mu.Lock()
	defer uds.mu.Unlock()

	dialog, ok := uds.dialogByUserID[userID]
	if !ok {
		return nil
	}
	return dialog
}

func (uds *UserDialogState) SetDialogForUser(userID int64, dialog Dialog) {
	uds.mu.Lock()
	defer uds.mu.Unlock()

	uds.dialogByUserID[userID] = dialog
}
