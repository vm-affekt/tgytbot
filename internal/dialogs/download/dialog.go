package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/dialogs/progress"
	"github.com/vm-affekt/tgytbot/internal/downloader"
	"github.com/vm-affekt/tgytbot/internal/logging"
)

const (
	btnStop   = "Прервать"
	btnStatus = "Статус"
)

var keyboardOnWait = tgbotapi.NewOneTimeReplyKeyboard(
	[]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton(btnStop),
		tgbotapi.NewKeyboardButton(btnStatus),
	},
)

const defaultAudioMaxFileSizeMB = 48

type dialog struct {
	rup                app.ReqUserProvider
	downloadService    app.DownloadService
	downloadingTimeout time.Duration
	audioMaxFileSize   int64

	statusMx             sync.Mutex
	isDownloadInProgress bool
	status               *downloadStatus
	messagesToDelete     messagesToDelete
}

type downloadStatus struct {
	title           string
	progressCounter *progress.Counter
	cancel          func()
}

type messagesToDelete struct {
	mu  sync.Mutex
	ids []int
}

func (mtd *messagesToDelete) addMessage(id int) {
	mtd.mu.Lock()
	defer mtd.mu.Unlock()
	mtd.ids = append(mtd.ids, id)
}

func (mtd *messagesToDelete) getIDs() []int {
	mtd.mu.Lock()
	defer mtd.mu.Unlock()
	return mtd.ids
}

func New(rup app.ReqUserProvider, downloadService app.DownloadService, downloadingTimeout time.Duration, audioMaxFileSizeMB int64) app.Dialog {
	var audioMaxFileSize int64
	if audioMaxFileSizeMB == 0 {
		audioMaxFileSize = megabytesToBytes(defaultAudioMaxFileSizeMB)
	} else {
		audioMaxFileSize = megabytesToBytes(audioMaxFileSizeMB)
	}
	return &dialog{
		rup:                rup,
		downloadService:    downloadService,
		downloadingTimeout: downloadingTimeout,
		audioMaxFileSize:   audioMaxFileSize,
	}
}

func (d *dialog) OnEnter(ctx context.Context) error {
	log := logging.FromContextS(ctx)
	log.Info("User entered to download dialog")
	return nil
}

func (d *dialog) OnMessage(ctx context.Context, text string, msgID int) error {
	if !d.isDownloading() {
		go d.startAudioDownloading(ctx, text)
	} else {
		d.messagesToDelete.addMessage(msgID)
		return d.onDownloading(ctx, text)
	}
	return nil
}

func (d *dialog) sendMsgWithKeyboardf(ctx context.Context, text string, vals ...interface{}) (err error) {
	_, err = d.rup.SendMessageWithKeyboardf(ctx, &keyboardOnWait, text, vals...)
	return err
}

func (d *dialog) sendMsgWithKeyboardThenDeletef(ctx context.Context, text string, vals ...interface{}) (err error) {
	msgID, err := d.rup.SendMessageWithKeyboardf(ctx, &keyboardOnWait, text, vals...)
	if err != nil {
		return err
	}
	d.messagesToDelete.addMessage(msgID)
	return nil
}

func (d *dialog) isDownloading() bool {
	d.statusMx.Lock()
	defer d.statusMx.Unlock()
	return d.isDownloadInProgress
}

func (d *dialog) printCurrentDownloadStatus(ctx context.Context) error {
	log := logging.FromContextS(ctx)
	log.Info("User requested progress status of downloading.")
	pc := d.status.progressCounter
	contentLen := pc.ContentLen()
	currentDownloaded := pc.CurrentDownloaded()
	contentLenMB, currentDownloadedMB := bytesToMegabytes(contentLen), bytesToMegabytes(currentDownloaded)
	if contentLen == 0 {
		return d.sendMsgWithKeyboardThenDeletef(ctx, "На данный момент загружено <b>%.2fMB</b>. Определить прогресс в процентах для данного видео невозможно...", currentDownloadedMB)
	}
	var estimatedTimeS string
	estimatedTime, err := pc.EstimatedTime()
	if err != nil {
		log.Warnf("Failed to count estimated time by reason: %v", err)
		estimatedTimeS = "???"
	} else {
		estimatedTime = estimatedTime.Round(time.Second)
		estimatedTimeS = fmt.Sprintf("%s", estimatedTime)
	}
	return d.sendMsgWithKeyboardThenDeletef(ctx, "На данный момент загружено\n<i>%.2fMB</i> из <i>%.2fMB</i>: <b>%.2f%%</b>\nПриблизительно осталось: <b>%s</b>", currentDownloadedMB, contentLenMB, pc.Percentage(), estimatedTimeS)
}

func (d *dialog) onDownloading(ctx context.Context, text string) error {
	if err := downloader.ValidateLink(text); err == nil {
		return d.sendMsgWithKeyboardThenDeletef(ctx, "Вы не можете скачивать другие видео/аудио, пока не завершится текущая загрузка! Вы можете ее отменить.")
	}
	if text == btnStop {
		if err := d.stopDownloading(ctx); err != nil {
			return fmt.Errorf("failed to stop downloading: %w", err)
		}
		return nil
	}
	if err := d.printCurrentDownloadStatus(ctx); err != nil {
		return fmt.Errorf("failed to print current download status: %w", err)
	}
	return nil
}

func (d *dialog) startAudioDownloading(ctx context.Context, link string) {
	log := logging.FromContextS(ctx)
	startT := time.Now()
	d.statusMx.Lock()
	d.isDownloadInProgress = true
	d.statusMx.Unlock()
	defer func() {
		log.Infof("Elapsed time of dowloading audio %q is %v", link, time.Since(startT).String())
		_, _ = d.rup.RedirectToDialog(ctx, app.DialogMain)
	}()
	if err := d.downloadAudio(ctx, link); err != nil {
		log.Errorf("Failed to download audio %q: %v", link, err)
		var textMsg string
		if d.status != nil {
			textMsg = fmt.Sprintf("При скачивании аудио из видео <b>%q</b> произошла техническая ошибка. Повторите попытку позже!", d.status.title)
		} else {
			textMsg = "При скачивании аудио из данного видео произошла техническая ошибка. Повторите попытку позже!"
		}
		textMsg += fmt.Sprintf("\n\nТекст ошибки:\n<code>%s</code>", err.Error())
		_, _ = app.SendMessagef(context.Background(), d.rup, textMsg) // TODO: КОД ОШИБКИ!
	}
}

func (d *dialog) downloadAudio(msgCtx context.Context, link string) error {
	ctx := logging.CopyContext(msgCtx, context.Background())
	var cancel func()
	if d.downloadingTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, d.downloadingTimeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	log := logging.FromContextS(ctx)
	log.Infof("Starting download audio by link: %q", link)
	downloadRes, err := d.downloadService.DownloadAudio(ctx, link)
	if err != nil {
		return fmt.Errorf("failed to download audio: %w", err)
	}
	progressCounter := progress.NewCounter(downloadRes.ContentLen)
	d.status = &downloadStatus{
		title:           downloadRes.Name,
		progressCounter: progressCounter,
		cancel:          cancel,
	}

	var partsCount int64

	isMultipart := true
	isKnownTotalSize := downloadRes.ContentLen > 0
	if isKnownTotalSize {
		partsCount = int64(math.Ceil(float64(downloadRes.ContentLen) / float64(d.audioMaxFileSize)))
		isMultipart = partsCount > 1
	}
	startMsg := "Загрузка аудио началась. Вы можете отменить или узнать статус загрузки, нажав соответствующие кнопки на клавиатуре."
	if isMultipart {
		if isKnownTotalSize {
			startMsg += fmt.Sprintf("\n\nИз-за ограничения Telegram для загрузки медиафайлов ботами, данное аудио будет разбито на <b>%d</b> частей.\nОни будут отправлены вам по мере готовности каждой отдельной записи.", partsCount)
		} else {
			startMsg += "\n\nИз-за ограничения Telegram для загрузки медиафайлов ботами, данное аудио может быть разбито на неопределенное количество частей, т.к у данного видео невозможно определить размер.\nОни будут отправлены вам по мере готовности каждой отдельной записи."
		}
	}
	if err := d.sendMsgWithKeyboardf(ctx, startMsg); err != nil {
		return err
	}
	audioStreamTee := io.TeeReader(downloadRes.Stream, d.status.progressCounter)

	partNum := 1
	for {
		ctx := logging.NewContextS(ctx,
			"part_num", partNum,
		)
		fileName := fmt.Sprintf("%s.mp3", downloadRes.Name)
		if isMultipart {
			fileName = fmt.Sprintf("p%d_", partNum) + fileName
		}
		log.Infof("Began to upload audio part %d...", partNum)
		audioUploadDone := make(chan error)
		pReader, pWriter := io.Pipe()
		go func() {
			if err := d.rup.SendAudio(ctx, pReader, fileName); err != nil {
				audioUploadDone <- fmt.Errorf("failed to send audio: %w", err)
			}
			audioUploadDone <- nil
		}()
		written, err := io.CopyN(pWriter, audioStreamTee, d.audioMaxFileSize)
		var lastPart bool
		if err != nil {
			if errors.Is(err, io.EOF) {
				lastPart = true
				log.Info("Got EOF from stream. It was a last part of audio.")
			} else {
				return fmt.Errorf("failed to copyN bytes to upload stream of part %d: %w", partNum, err)
			}
		}
		log.Infof("Copied %d bytes (%.2f MB) to pipe writer. Waiting for upload done...", written, bytesToMegabytes(written))
		if err := pWriter.Close(); err != nil {
			return fmt.Errorf("failed to close pipe writer for uploader of part %d: %w", partNum, err)
		}
		if err := <-audioUploadDone; err != nil {
			return fmt.Errorf("failed to sendAudio of part %d: %w", partNum, err)
		}
		if isMultipart {
			if isKnownTotalSize {
				if err := d.sendMsgWithKeyboardf(ctx, `<b>%d/%d</b> часть вашего аудио успешно загружена!`, partNum, partsCount); err != nil {
					return err
				}
			} else {
				if err := d.sendMsgWithKeyboardf(ctx, "<b>%d</b> часть вашего аудио успешно загружена!", partNum); err != nil {
					return err
				}
			}
		}
		log.Info("Audio part upload done successfully!")
		if lastPart {
			break
		}

		partNum++
	}

	log.Info("Successfully downloaded!")
	if _, err := app.SendMessagef(ctx, d.rup, "Аудио из видео <b>%q</b> успешно и полностью загружено!", d.status.title); err != nil {
		return err
	}
	go func() {
		if err := d.clearMessages(ctx); err != nil {
			log.Errorf("Failed to delete messages: %v", err)
		}
	}()
	return nil
}

func (d *dialog) clearMessages(ctx context.Context) error {
	return d.rup.DeleteMessages(ctx, d.messagesToDelete.getIDs()...)
}

func (d *dialog) stopDownloading(ctx context.Context) error {
	log := logging.FromContextS(ctx)
	log.Info("User requested to stop downloading!")
	d.status.cancel()
	if _, err := app.SendMessagef(ctx, d.rup, "Вы успешно прервали загрузку."); err != nil {
		return err
	}
	if _, err := d.rup.RedirectToDialog(ctx, app.DialogMain); err != nil {
		return err
	}
	return nil
}

const oneMB = 1048576

func bytesToMegabytes(bytes int64) float64 {
	return float64(bytes) / float64(oneMB)
}

func megabytesToBytes(mbs int64) int64 {
	return mbs * oneMB
}
