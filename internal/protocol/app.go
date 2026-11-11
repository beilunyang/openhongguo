package protocol

// Public application constants observed on com.phoenix.read 7.3.9.32.
// They are app identity, not a copied device credential.
const (
	Package        = "com.phoenix.read"
	VersionName    = "7.3.9.32"
	VersionCode    = "73932"
	AID            = "8662"
	AppName        = "novelread"
	DevicePlatform = "android"

	// APIHost is the host the captured client sends every /reading/ and
	// /novel/ request to (1508 of 1508, 2026-10-24). reading.snssdk.com
	// answers the same reads but 404s the distribution and activity families.
	APIHost = "api5-normal-sinfonlinea.fqnovel.com"
	// RegisterHost is where the captured client registers its device
	// (2026-10-24). UrlConfig.CHINA's log.snssdk.com registers devices the
	// same way.
	RegisterHost = "log5-applog-lq.fqnovel.com"

	PathDiscover   = "/reading/bookapi/bookmall/tab/v"
	PathCellChange = "/reading/bookapi/bookmall/cell/change/v"
	PathSearch     = "/reading/bookapi/search/tab/v"
	PathSeries     = "/novel/player/video_detail/v1/"
	// PathSeriesBatch is the player's batch detail read (multi_video_detail,
	// player scene; observed 2026-10-24).
	PathSeriesBatch     = "/novel/player/multi_video_detail/player/v1"
	PathPlayback        = "/novel/player/video_model/v1/"
	PathAlbum           = "/novel/player/album_detail/v1/"
	PathSuggest         = "/reading/bookapi/search/suggest/v1/"
	PathCue             = "/reading/bookapi/search/cue/v1/"
	PathCategory        = "/reading/bookapi/new_category/front/v1/"
	PathCategoryLanding = "/reading/bookapi/new_category/landing/v1/"
	PathRank            = "/reading/ugc/rank_book/get/v1/"
	PathPlan            = "/reading/bookapi/plan/v1/"
	PathRankActivity    = "/novel/reading/activity_api/get_rank_activity/v1/"
	PathRankCells       = "/reading/bookapi/bookmall/cell/change/v1/"
	// PathSubscribeList serves the 上新 calendar (tab_type 5); PathProfile
	// and PathUserWorks a person's page and their works (observed 2026-10-24).
	PathSubscribeList = "/reading/user/subscribe/list/v1/"
	PathProfile       = "/reading/user/profile/get/v1/"
	PathUserWorks     = "/novel/commentapi/user/profile/%s/v1"
	// PathDanmaku is the comment list the player's 弹幕 layer reads; %s is
	// the episode id (observed 2026-11-11).
	PathDanmaku = "/novel/commentapi/comment/list/%s/v1/"
	// DefaultRankCell is the bookmall cell the 经典 channel serves for the
	// ranking page (observed: algo 179 推荐榜).
	DefaultRankCell = "7224069876808155173"
	// ShortRankCell is the bookmall cell behind the short-drama ranking page
	// (tab_type 26); cell/change on it serves every content and list.
	ShortRankCell = "7470092475068071998"
	// PathShortFilter is the short-drama filter page (筛选). The reading host
	// answers 404; it lives on DistributionHost.
	PathShortFilter = "/reading/distribution/category/landpage/v1/"
	// DistributionHost serves the /reading/distribution/ and activity_api
	// families. It is APIHost; it stays separate so APIOrigin can point at
	// reading.snssdk.com, which answers 404 for them.
	DistributionHost = APIHost
	// DefaultRankActivity and DefaultRankList are the hard-coded ids of the
	// 星粉榜 activity in the shipped client.
	DefaultRankActivity = "hongguo_actor_fans_ranklist_activity"
	DefaultRankList     = "hongguo_actor_fans_ranklist"
	PathRegister        = "/service/2/device_register/"

	SearchTabShortPlay = "11"
	// Search tabs filled by an anonymous search response.
	// SeriesVideo is 剧集, PugcVideo is 视频, Discussion is 讨论,
	// Common is 小说, HgUser is 用户, and Voice is 听书.
	SearchTabSeriesVideo = "29"
	SearchTabPugcVideo   = "30"
	SearchTabDiscussion  = "31"
	SearchTabCommon      = "1"
	SearchTabHgUser      = "27"
	SearchTabVoice       = "2"
	// SearchSourceHome is the home-search first page (search_source=1, tab_name=feed).
	SearchSourceHome = "1"
	SearchTabName    = "feed"
)

// CredentialHeaderNames are account credentials. Anonymous search does not send them.
// A guest cookie from this installation and lc=101 are not account credentials.
var CredentialHeaderNames = []string{
	"Authorization",
	"X-Tt-Token",
	"Openid",
}
