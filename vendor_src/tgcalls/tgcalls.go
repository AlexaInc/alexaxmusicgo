package tgcalls

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"fmt"
	"github.com/amarnathcjd/tgcalls/ntgcalls"
	"github.com/amarnathcjd/gogram/telegram"
)

type GroupCall struct {
	client *telegram.Client
	ntg    *ntgcalls.Client
}

func NewGroupCall(client *telegram.Client) *GroupCall {
	return &GroupCall{
		client: client,
		ntg:    ntgcalls.NTgCalls(),
	}
}

func (g *GroupCall) Free() {
	g.ntg.Free()
}

func (g *GroupCall) Play(chatID int64, params *MediaParams) error {
	desc := ntgcalls.MediaDescription{
		Microphone: &ntgcalls.AudioDescription{
			MediaSource:  ntgcalls.MediaSourceShell,
			SampleRate:   128000,
			ChannelCount: 2,
			Input:        fmt.Sprintf("ffmpeg -i %s -loglevel panic -f s16le -ac 2 -ar 128k pipe:1", params.Path),
		},
	}
	if params.Video {
		desc.Video = &ntgcalls.VideoDescription{
			InputMode: ntgcalls.InputModeShell,
			Input:     fmt.Sprintf("ffmpeg -i %s -loglevel panic -f rawvideo -r 24 -pix_fmt yuv420p -vf scale=1280:720 pipe:1", params.Path),
			Width:     1280,
			Height:    720,
			Fps:       24,
		}
	}

	jsonParams, err := g.ntg.CreateCall(chatID, desc)
	if err != nil {
		return err
	}

	call, err := g.client.GetGroupCall(chatID)
	if err != nil {
		return err
	}

	me, _ := g.client.GetMe()
	_, err = g.client.PhoneJoinGroupCall(
		&telegram.PhoneJoinGroupCallParams{
			Muted:        false,
			VideoStopped: !params.Video,
			Call:         call,
			Params: &telegram.DataJson{
				Data: jsonParams,
			},
			JoinAs: &telegram.InputPeerUser{
				UserID:     me.ID,
				AccessHash: me.AccessHash,
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (g *GroupCall) LeaveCall(chatID int64) error {
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
		f(chatID)
	})
}

func (g *GroupCall) OnLeave(f func(int64)) {
	// Not directly supported by ntgcalls? We'll use OnStreamEnd logic or similar
}

type MediaParams struct {
	Path       string
	Audio      bool
	Video      bool
	SeekDelay  int
	Headers    map[string]string
	FFmpegArgs string
}

func joinGroupCall(ntg *ntgcalls.Client, client *telegram.Client, username string, url string) {
	me, _ := client.GetMe()
	rawChannel, _ := client.ResolveUsername(username)
	channel := rawChannel.(*telegram.Channel)
	jsonParams, _ := ntg.CreateCall(channel.ID, ntgcalls.MediaDescription{
		Microphone: &ntgcalls.AudioDescription{
			MediaSource:  ntgcalls.MediaSourceShell, // ntgcalls.MediaSourceFile
			SampleRate:   128000,                    // 96000
			ChannelCount: 2,
			Input:        fmt.Sprintf("ffmpeg -i %s -loglevel panic -f s16le -ac 2 -ar 128k pipe:1", url), // './file.s16le'
		},
	})
	call, err := client.GetGroupCall(channel.ID)
	if err != nil {
		panic(err)
	}

	callResRaw, _ := client.PhoneJoinGroupCall(
		&telegram.PhoneJoinGroupCallParams{
			Muted:        false,
			VideoStopped: true,
			Call:         call,
			Params: &telegram.DataJson{
				Data: jsonParams,
			},
			JoinAs: &telegram.InputPeerUser{
				UserID:     me.ID,
				AccessHash: me.AccessHash,
			},
		},
	)
	callRes := callResRaw.(*telegram.UpdatesObj)
	for _, update := range callRes.Updates {
		switch update := update.(type) {
		case *telegram.UpdateGroupCallConnection:
			phoneCall := update
			_ = ntg.Connect(channel.ID, phoneCall.Params.Data, false)
		}
	}
}

func outgoingCall(client *ntgcalls.Client, mtproto *telegram.Client, username string) {
	var inputCall *telegram.InputPhoneCall

	rawUser, _ := mtproto.ResolveUsername(username)
	user := rawUser.(*telegram.UserObj)
	dhConfigRaw, _ := mtproto.MessagesGetDhConfig(0, 256)
	dhConfig := dhConfigRaw.(*telegram.MessagesDhConfigObj)
	_ = client.CreateP2PCall(user.ID, ntgcalls.MediaDescription{
		Microphone: &ntgcalls.AudioDescription{
			MediaSource:  ntgcalls.MediaSourceShell,
			SampleRate:   96000,
			ChannelCount: 2,
			Input:        "ffmpeg -reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2 -i https://docs.evostream.com/sample_content/assets/sintel1m720p.mp4 -f s16le -ac 2 -ar 96k -v quiet pipe:1",
		},
	})
	gAHash, _ := client.InitExchange(user.ID, ntgcalls.DhConfig{
		G:      dhConfig.G,
		P:      dhConfig.P,
		Random: dhConfig.Random,
	}, nil)
	protocolRaw := ntgcalls.GetProtocol()
	protocol := &telegram.PhoneCallProtocol{
		UdpP2P:          protocolRaw.UdpP2P,
		UdpReflector:    protocolRaw.UdpReflector,
		MinLayer:        protocolRaw.MinLayer,
		MaxLayer:        protocolRaw.MaxLayer,
		LibraryVersions: protocolRaw.Versions,
	}
	_, _ = mtproto.PhoneRequestCall(
		&telegram.PhoneRequestCallParams{
			Protocol: protocol,
			UserID:   &telegram.InputUserObj{UserID: user.ID, AccessHash: user.AccessHash},
			GAHash:   gAHash,
			RandomID: int32(telegram.GenRandInt()),
		},
	)

	mtproto.AddRawHandler(&telegram.UpdatePhoneCall{}, func(m telegram.Update, c *telegram.Client) error {
		phoneCall := m.(*telegram.UpdatePhoneCall).PhoneCall
		switch phoneCall.(type) {
		case *telegram.PhoneCallAccepted:
			call := phoneCall.(*telegram.PhoneCallAccepted)
			res, _ := client.ExchangeKeys(user.ID, call.GB, 0)
			inputCall = &telegram.InputPhoneCall{
				ID:         call.ID,
				AccessHash: call.AccessHash,
			}
			client.OnSignal(func(chatId int64, signal []byte) {
				_, _ = mtproto.PhoneSendSignalingData(inputCall, signal)
			})
			callConfirmRes, _ := mtproto.PhoneConfirmCall(
				inputCall,
				res.GAOrB,
				res.KeyFingerprint,
				protocol,
			)
			callRes := callConfirmRes.PhoneCall.(*telegram.PhoneCallObj)
			rtcServers := make([]ntgcalls.RTCServer, len(callRes.Connections))
			for i, connection := range callRes.Connections {
				switch connection := connection.(type) {
				case *telegram.PhoneConnectionWebrtc:
					rtcServer := connection
					rtcServers[i] = ntgcalls.RTCServer{
						ID:       rtcServer.ID,
						Ipv4:     rtcServer.Ip,
						Ipv6:     rtcServer.Ipv6,
						Username: rtcServer.Username,
						Password: rtcServer.Password,
						Port:     rtcServer.Port,
						Turn:     rtcServer.Turn,
						Stun:     rtcServer.Stun,
					}
				case *telegram.PhoneConnectionObj:
					phoneServer := connection
					rtcServers[i] = ntgcalls.RTCServer{
						ID:      phoneServer.ID,
						Ipv4:    phoneServer.Ip,
						Ipv6:    phoneServer.Ipv6,
						Port:    phoneServer.Port,
						Turn:    true,
						Tcp:     phoneServer.Tcp,
						PeerTag: phoneServer.PeerTag,
					}
				}
			}
			_ = client.ConnectP2P(user.ID, rtcServers, callRes.Protocol.LibraryVersions, callRes.P2PAllowed)
		}
		return nil
	})

	mtproto.AddRawHandler(&telegram.UpdatePhoneCallSignalingData{}, func(m telegram.Update, c *telegram.Client) error {
		signalingData := m.(*telegram.UpdatePhoneCallSignalingData).Data
		_ = client.SendSignalingData(user.ID, signalingData)
		return nil
	})
}
