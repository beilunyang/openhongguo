package protocol

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strconv"

	"github.com/beilunyang/openhongguo/sdk/model"
)

const (
	PathPlaybackV5      = "/novel/player/multi_video_model/player/v1"
	PathVideoTab        = "/reading/bookapi/video_tab/multi_video_model/v1/"
	VideoTabUserAgent   = "com.phoenix.read/73932 (Linux; U; Android 16; zh_CN; 25053RT47C; Build/BP2A.250605.031.A3; Cronet/TTNetVersion:04657795 2026-01-23 QuicVersion:c67e9834 2025-09-08)"
	PlaybackVersionCode = 74332
	PlaybackVersionName = "7.4.3.32"
)

// PlaybackUserAgent is the Cronet User-Agent the 7.4.3.32 client sends with
// the V5 playback request: UserAgent's format with the playback version.
func PlaybackUserAgent(d model.Device) string {
	d = d.OrDefault()
	return fmt.Sprintf("%s/%d (Linux; U; Android %s; zh_CN; %s; Build/%s; Cronet/TTNetVersion:8d40f833 2026-03-03 QuicVersion:21ac1950 2025-11-18)",
		Package, PlaybackVersionCode, d.OSVersion, d.Model, d.Build)
}

// PlaybackV5Pairs is the V5 playback query in the order the 7.4.3.32 client
// sends it (captured 2026-11-18), with the profile's device and anonymous
// install. The empty _rticket marks where the signer puts the ticket; order
// is part of the signature.
func PlaybackV5Pairs(id SearchIdentity) [][2]string {
	d := id.Device.OrDefault()
	kind, pad := "phone", "0"
	if d.Pad {
		kind, pad = "pad", "1"
	}
	version := strconv.Itoa(PlaybackVersionCode)
	return [][2]string{
		{"device_platform", DevicePlatform},
		{"os", "android"},
		{"ssmix", "a"},
		{"_rticket", ""},
		{"cdid", id.CDID},
		{"channel", d.Channel},
		{"aid", AID},
		{"app_name", AppName},
		{"version_code", version},
		{"version_name", PlaybackVersionName},
		{"manifest_version_code", version},
		{"update_version_code", version},
		{"resolution", d.Resolution},
		{"dpi", strconv.Itoa(d.DPI)},
		{"device_type", d.Model},
		{"device_brand", d.Brand},
		{"language", "zh"},
		{"os_api", strconv.Itoa(d.OSAPI)},
		{"os_version", d.OSVersion},
		{"ac", "wifi"},
		{"host_abi", "arm64-v8a"},
		{"dragon_device_type", kind},
		{"pv_player", version},
		{"compliance_status", "0"},
		{"need_personal_recommend", "1"},
		{"player_so_load", "1"},
		{"is_android_pad_screen", pad},
		{"iid", id.InstallID},
		{"device_id", id.DeviceID},
	}
}

// VideoTabPairs is the bodyless video_tab query used when the V5 player
// endpoint answers an empty model. The fixed phone template matches the
// upstream 7.4.3.32 client; only the anonymous identity and episode vary.
func VideoTabPairs(episodeID string, id SearchIdentity) [][2]string {
	d := id.Device.OrDefault()
	return [][2]string{
		{"aid", AID},
		{"app_name", AppName},
		{"version_code", strconv.Itoa(PlaybackVersionCode)},
		{"version_name", PlaybackVersionName},
		{"channel", d.Channel},
		{"device_platform", "android"},
		{"device_type", d.Model},
		{"os_version", d.OSVersion},
		{"update_version_code", strconv.Itoa(PlaybackVersionCode)},
		{"iid", id.InstallID},
		{"cdid", id.CDID},
		{"device_id", id.DeviceID},
		{"video_id", episodeID},
		{"video_id_type", "0"},
		{"device_level", "1"},
		{"need_all_video_definition", "1"},
		{"need_mp4_align", "0"},
		{"use_os_player", "0"},
		{"video_platform", "1024"},
		{"source", "0"},
		{"from_video_id", ""},
	}
}

// PlaybackV5Body 只压缩一次；调用方将返回的原始字节同时用于签名和发送。
//
// device_level 保持 1，和 App 的 3 不同：3 会让服务端把 1080p 以下的码流
// 都换成 bytevc2（2026-11-18 实测），App 自带解码器，ffmpeg 和 mpv 都解不了；
// 1 下发的全是 bytevc1。
func PlaybackV5Body(episodeID string) ([]byte, error) {
	body, err := jsonMarshal(map[string]any{
		"biz_param": map[string]any{
			"detail_page_version": 0, "device_level": 1, "disable_digg_stat": false,
			"disable_video_relate_book": false, "need_all_video_definition": true,
			"need_mp4_align": false, "use_os_player": false, "use_server_dns": false, "video_platform": 1024,
		},
		"mixed_video_id_map": map[string][]string{"1004": {episodeID}},
		"dr_scene":           "player",
	})
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	writer := gzip.NewWriter(&output)
	if _, err = writer.Write(body); err != nil {
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
