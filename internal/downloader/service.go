package downloader

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"

	"github.com/kkdai/youtube/v2"
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/logging"
	"go.uber.org/zap"
)

type Service struct {
	debugMode bool
}

func New(debugMode bool) *Service {
	return &Service{
		debugMode: debugMode,
	}
}

func (s *Service) DownloadAudio(ctx context.Context, link string) (result app.DownloadResult, err error) {
	const audioMP4PatternMime = "audio/mp4"
	ctx = logging.NewContextS(ctx, zap.String("video_link", link))

	mp4DownloadRes, err := s.downloadStream(ctx, link, audioMP4PatternMime)
	if err != nil {
		return app.DownloadResult{}, fmt.Errorf("failed to start downloading stream: %w", err)
	}

	mp3Stream, err := s.convertMP4ToMP3(ctx, mp4DownloadRes.Stream)
	if err != nil {
		return app.DownloadResult{}, fmt.Errorf("failed to convert from mp4 to mp3: %w", err)
	}
	return app.DownloadResult{
		ContentLen: mp4DownloadRes.ContentLen,
		Name:       mp4DownloadRes.Name,
		Stream:     mp3Stream,
	}, nil

}

func (s *Service) DownloadVideo(ctx context.Context, link string) (app.DownloadResult, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Service) downloadStream(ctx context.Context, link string, formatType string) (result app.DownloadResult, err error) {
	log := logging.FromContextS(ctx)
	ytClient := &youtube.Client{
		Debug: s.debugMode,
	}

	link = s.transformLink(ctx, link)
	video, err := ytClient.GetVideo(link)
	if err != nil {
		return app.DownloadResult{}, fmt.Errorf("failed to get video by link: %w", err)
	}
	log.Infof("Got video metadata with %d formats", len(video.Formats))
	formats := video.Formats.WithAudioChannels().Type(formatType)
	if len(formats) == 0 {
		return app.DownloadResult{}, fmt.Errorf("no video format found for type pattern %q", formatType)
	}
	format := &formats[0]
	log.Infow("Found video format for pattern "+formatType,
		"format_url", format.URL,
		"format_mime_type", format.MimeType,
		"format_quality", format.Quality,
		"format_itag", format.ItagNo,
	)
	stream, contentLen, err := ytClient.GetStreamContext(ctx, video, format)
	if err != nil {
		return app.DownloadResult{}, fmt.Errorf("failed to get video stream: %w", err)
	}
	log.Infof("Started downloading stream. Content length is %d", contentLen)

	return app.DownloadResult{
		ContentLen: contentLen,
		Name:       video.Title,
		Stream:     stream,
	}, err
}

func (s *Service) convertMP4ToMP3(ctx context.Context, mp4Stream io.ReadCloser) (mp3Stream io.ReadCloser, err error) {
	log := logging.FromContextS(ctx)
	log.Info("Converting from MP4 to MP3 via ffmpeg...")
	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", "-i", "pipe:", "-f", "mp3", "-")
	ffmpegCmd.Stdin = mp4Stream

	mp3Stream, err = ffmpegCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get ffmpeg stdout pipe: %w", err)
	}
	if err := ffmpegCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg cmd: %w", err)
	}
	log.Info("ffmpeg converter started! Waiting...")
	go func() {
		defer mp4Stream.Close()
		// TODO: В доке пишут что Wait не нужно вызывать до того, как все прочитают из Reader, но вроде работает все
		if err := ffmpegCmd.Wait(); err != nil {
			log.Errorf("ffmpeg: An error occurred while Wait: %v", err)
		}
		log.Info("ffmpeg converter done!")
	}()

	return mp3Stream, nil
}

// transformLink extracts and returns video id if link has '/live/' path.
// Youtube downloader lib has bug: it doesn't recognize '/live/' links.
func (s *Service) transformLink(ctx context.Context, link string) string {
	const livePath = "/live/"
	log := logging.FromContextS(ctx)
	parsedURL, err := url.Parse(link)
	if err != nil {
		log.Errorf("downloader.transformLink: failed to parse url: %v", err)
		return link
	}
	path := parsedURL.Path
	if !strings.HasPrefix(path, livePath) {
		return link
	}
	startIdx := len(livePath)
	if len(path) == startIdx {
		log.Errorf("downloader.transformLink: no video_id after %s", livePath)
		return link
	}
	return path[startIdx:]
}
