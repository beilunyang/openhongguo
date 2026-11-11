// Package fixtures 用 testdata/protocol 的样本模拟红果的接口，供各语言绑定的
// 测试使用，不访问网络。
//
// 路径以 /limited/ 开头的请求一律返回 429，用来测试限流的错误映射。
package fixtures

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// routes 把接口路径映射到样本文件。
var routes = map[string]string{
	"/service/2/device_register/":                         "register-response.json",
	"/reading/bookapi/bookmall/tab/v":                     "discover-response.json",
	"/reading/bookapi/search/tab/v":                       "search-response.json",
	"/novel/player/video_detail/v1/":                      "series-response.json",
	"/novel/player/video_model/v1/":                       "drm-playback.json",
	"/novel/commentapi/comment/list/7100000000000201/v1/": "danmaku-response.json",
}

// Handler 返回回放 dir（即 testdata/protocol）下样本的 HTTP 处理器。
func Handler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/limited/") {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		name, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
}
