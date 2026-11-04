package sdk

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// Categories 加载书籍分类的标签页：全部、男生、女生和默认标签页。
// 线上服务在这里不返回行（2026-10-24 实测）；列表及其筛选来自 CategoryLanding。
//
// 路径和请求类取自 APK：GetNewCategoryFrontPageRequest 发往
// /reading/bookapi/new_category/front/v:version/。
func (c *Client) Categories(ctx context.Context, categoryID string) (*model.CategoryPage, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.CategoryPage, error) {
		if strings.TrimSpace(categoryID) == "" {
			return nil, fmt.Errorf("missing category ID")
		}
		const offset = 0
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignGET(protocol.SignedCategoryPairs(c.identity(p), categoryID, offset, 20), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathCategory+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		return protocol.ParseCategoryFront(raw)
	})
}

// LandingOptions 选择一个分类落地页。
//
// Tab 是 Categories 给出的 tab 值：8（全部，默认）、1（男生）或 0（女生）。
// Filters 把 CategoryPage.Filters 中的筛选组键（book_status、sort_by、
// sub_category、word_type）映射到该组的某个选项值。
// Cursor 是上一页的 NextOffset。
type LandingOptions struct {
	Tab     string
	Filters map[string]string
	Cursor  string
}

// CategoryLanding 加载书籍分类下列表的一页，以及它的筛选面板。
// 这些行是书（小说），不是短剧。
func (c *Client) CategoryLanding(ctx context.Context, categoryID string, opts LandingOptions) (*model.CategoryPage, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.CategoryPage, error) {
		if strings.TrimSpace(categoryID) == "" {
			return nil, fmt.Errorf("missing category ID")
		}
		gender := ""
		switch opts.Tab {
		case "", "8":
		case "1", "0":
			// tab 以 query_gender 发送；tab_type 不起作用（实测）。
			gender = opts.Tab
		default:
			return nil, fmt.Errorf("unknown category tab %q; use 8, 1, or 0", opts.Tab)
		}
		for key := range opts.Filters {
			if !slices.ContainsFunc(protocol.LandingFilterParams, func(kv [2]string) bool { return kv[0] == key }) {
				return nil, fmt.Errorf("unknown category filter %q", key)
			}
		}
		offset, err := protocol.SearchOffset(opts.Cursor)
		if err != nil {
			return nil, err
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignGET(protocol.SignedCategoryLandingPairs(c.identity(p), categoryID, gender, opts.Filters, offset, 20), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathCategoryLanding+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		return protocol.ParseCategoryLanding(raw)
	})
}
