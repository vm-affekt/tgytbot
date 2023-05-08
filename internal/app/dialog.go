package app

import (
	"context"
	"fmt"
)

// Dialog is the interface for bot's dialogs. Implementations should be in the 'dialogs' directory.
type Dialog interface {
	// OnEnter called when user enters this dialog
	OnEnter(ctx context.Context) error
	// OnMessage called when user sends a message, being in this dialog
	OnMessage(ctx context.Context, text string, msgID int) error
}

type DialogID int

const (
	DialogMain = DialogID(iota)
	DialogYoutubeDownload
)

var allDialogIDs = map[DialogID]struct{}{
	DialogMain:            {},
	DialogYoutubeDownload: {},
}

func (id DialogID) Validate() error {
	_, ok := allDialogIDs[id]
	if !ok {
		return fmt.Errorf("%v is unknown dialog id", id)
	}
	return nil
}
