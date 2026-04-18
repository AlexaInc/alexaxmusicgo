package tgcalls

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/amarnathcjd/tgcalls/ntgcalls"
	"github.com/amarnathcjd/gogram/telegram"
)

type GroupCall struct {
	client   *telegram.Client
	ntg      *ntgcalls.Client
	mu       sync.RWMutex
	inCall   map[int64]bool // chatID → already joined
}

func NewGroupCall(client *telegram.Client) *GroupCall {
	return &GroupCall{
		client: client,
		ntg:    ntgcalls.NTgCalls(),
		inCall: make(map[int64]bool),
	}
}

func (g *GroupCall) Free() {
	g.ntg.Free()
}

func (g *GroupCall) Play(chatID int64, params *MediaParams) error {
	desc := buildDesc(params)

	g.mu.RLock()
	already := g.inCall[chatID]
	g.mu.RUnlock()

	if already {
		// Already in VC — just change the stream source without rejoining
		log.Printf("[tgcalls] ChangeStream chatID=%d", chatID)
		return g.ntg.SetStreamSources(chatID, ntgcalls.CaptureStream, desc)
	}

	// First time joining — create call, join, then connect
	log.Printf("[tgcalls] JoinCall chatID=%d", chatID)
	jsonParams, err := g.ntg.CreateCall(chatID, desc)
	if err != nil {
		return fmt.Errorf("CreateCall: %w", err)
	}

	call, err := g.client.GetGroupCall(chatID)
	if err != nil {
		return fmt.Errorf("GetGroupCall: %w", err)
	}

	me, err := g.client.GetMe()
	if err != nil {
		return fmt.Errorf("GetMe: %w", err)
	}

	res, err := g.client.PhoneJoinGroupCall(
		&telegram.PhoneJoinGroupCallParams{
			Muted:        false,
			VideoStopped: !params.Video,
			Call:         *call,
			Params:       &telegram.DataJson{Data: jsonParams},
			JoinAs: &telegram.InputPeerUser{
				UserID:     me.ID,
				AccessHash: me.AccessHash,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("PhoneJoinGroupCall: %w", err)
	}

	// Extract UpdateGroupCallConnection from the response and connect
	if updatesObj, ok := res.(*telegram.UpdatesObj); ok {
		for _, upd := range updatesObj.Updates {
			if conn, ok2 := upd.(*telegram.UpdateGroupCallConnection); ok2 {
				if err2 := g.ntg.Connect(chatID, conn.Params.Data, false); err2 != nil {
					return fmt.Errorf("Connect: %w", err2)
				}
				log.Printf("[tgcalls] Connected chatID=%d", chatID)
				g.mu.Lock()
				g.inCall[chatID] = true
				g.mu.Unlock()
				return nil
			}
		}
	}

	// If we get here the updates didn't contain the connection params — leave and error
	_ = g.ntg.Stop(chatID)
	return fmt.Errorf("PhoneJoinGroupCall: no UpdateGroupCallConnection in response")
}

func (g *GroupCall) LeaveCall(chatID int64) error {
	g.mu.Lock()
	delete(g.inCall, chatID)
	g.mu.Unlock()
	return g.ntg.Stop(chatID)
}

func (g *GroupCall) Pause(chatID int64) error {
	_, err := g.ntg.Pause(chatID)
	return err
}

func (g *GroupCall) Resume(chatID int64) error {
	_, err := g.ntg.Resume(chatID)
	return err
}

func (g *GroupCall) OnStreamEnd(f func(int64)) {
	g.ntg.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, streamDevice ntgcalls.StreamDevice) {
		// Only trigger PlayNext when the AUDIO (microphone) stream ends.
		// Ignoring CameraStream / ScreenStream endings which fire immediately
		// when no video source is configured.
		if streamDevice == ntgcalls.MicrophoneStream {
			f(chatID)
		}
	})
}

func (g *GroupCall) OnLeave(f func(int64)) {
	// handled via Stop / OnStreamEnd
}

// buildDesc constructs a MediaDescription from MediaParams.
func buildDesc(params *MediaParams) ntgcalls.MediaDescription {
	path := params.Path
	if !strings.HasPrefix(path, "http") {
		path = `"` + path + `"`
	}

	headers := ""
	if len(params.Headers) > 0 {
		hStr := ""
		for k, v := range params.Headers {
			hStr += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		headers = fmt.Sprintf("-headers \"%s\"", hStr)
	}

	isStream := strings.HasPrefix(params.Path, "http")
	inputFlags := "-threads 0"
	if isStream {
		inputFlags += " -reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 5"
	}
	if headers != "" {
		inputFlags += " " + headers
	}

	// Simplified FFmpeg command: Removing -re and aresample as suggested by user to fix cracking
	// Switching to Stereo 44.1kHz as the target format
	audioInput := fmt.Sprintf(
		"ffmpeg %s -i %s -vn -sn -loglevel warning -f s16le -ac 2 -ar 44100 pipe:1",
		inputFlags, path,
	)
 
	// Optimization: If file is already pre-transcoded PCM, use zero-CPU command
	if strings.HasSuffix(params.Path, ".pcm.raw") {
		audioInput = fmt.Sprintf(
			"ffmpeg -f s16le -ac 2 -ar 44100 -i %s -f s16le -ac 2 -ar 44100 pipe:1",
			path,
		)
	}
 
	if params.SeekDelay > 0 {
		audioInput = fmt.Sprintf(
			"ffmpeg %s -ss %d -i %s -vn -sn -loglevel warning -f s16le -ac 2 -ar 44100 pipe:1",
			inputFlags, params.SeekDelay, path,
		)
	}
	desc := ntgcalls.MediaDescription{
		Microphone: &ntgcalls.AudioDescription{
			MediaSource:  ntgcalls.MediaSourceShell,
			SampleRate:   44100,
			ChannelCount: 2, // Stereo
			Input:        audioInput,
		},
	}
	if params.Video {
		desc.Camera = &ntgcalls.VideoDescription{
			MediaSource: ntgcalls.MediaSourceShell,
			Input: fmt.Sprintf(
				"ffmpeg %s -i %s -loglevel warning -f rawvideo -r 24 -pix_fmt yuv420p -vf scale=1280:720 pipe:1",
				inputFlags, path,
			),
			Width:  1280,
			Height: 720,
			Fps:    24,
		}
	}
	return desc
}

type MediaParams struct {
	Path       string
	Audio      bool
	Video      bool
	SeekDelay  int
	Headers    map[string]string
	FFmpegArgs string
}
