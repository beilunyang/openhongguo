package protocol

import (
	"bytes"
	"compress/gzip"
	"strconv"
)

const (
	PathPlaybackV5      = "/novel/player/multi_video_model/player/v1"
	PlaybackVersionCode = 74332
	PlaybackVersionName = "7.4.3.32"
	// UA 保留 V5 协议模板原值，与查询串中的版本号分别处理。
	PlaybackUserAgent = "com.phoenix.read/73932 (Linux; U; Android 12; zh_CN_#Hans; PGT-AN10; Build/V417IR;tt-ok/3.12.13.20)"
)

// PlaybackV5Pairs 使用播放专用的手机端模板和当前匿名安装，顺序参与签名。
func PlaybackV5Pairs(id SearchIdentity) [][2]string {
	return [][2]string{
		{"aid", AID}, {"app_name", AppName},
		{"version_code", strconv.Itoa(PlaybackVersionCode)}, {"version_name", PlaybackVersionName},
		{"channel", "vivo_8662_64"}, {"device_platform", "android"},
		{"device_type", "PGT-AN10"}, {"os_version", "12"}, {"update_version_code", strconv.Itoa(PlaybackVersionCode)},
		{"iid", id.InstallID}, {"cdid", id.CDID}, {"device_id", id.DeviceID},
	}
}

// PlaybackV5Body 只压缩一次；调用方将返回的原始字节同时用于签名和发送。
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
