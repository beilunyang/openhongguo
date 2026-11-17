package sdk

import (
	"context"
	"fmt"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// Person 加载一位演员或一个角色的主页。userID 是账号 id（演员榜上的
// RankItem.UserID），不是演员 id。
func (c *Client) Person(ctx context.Context, userID string) (*model.Person, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Person, error) {
		if !validID(userID) {
			return nil, fmt.Errorf("invalid user ID")
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignGET(protocol.SignedProfilePairs(c.identity(p), userID), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathProfile+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		person, err := protocol.ParseProfile(raw)
		if err != nil {
			return nil, err
		}
		if person.Name == "" {
			return nil, ErrContentUnavailable
		}
		return person, nil
	})
}

// PersonWorks 加载一位演员参演的短剧，一次一页（主页上的剧作）。
// cursor 是上一页的 Cursor。
//
// 不同设备拿到的主页 tab 编号不一样（作品 › 剧作，或者剧作 › 全部），
// 所以第一页先像 App 一样取一次 tab 列表，找到剧作所在的 tab，再取作品；
// 后续页沿用游标里记下的 tab。主页没有剧作时返回空页。
func (c *Client) PersonWorks(ctx context.Context, userID, cursor string) (*model.WorkPage, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.WorkPage, error) {
		if !validID(userID) {
			return nil, fmt.Errorf("invalid user ID")
		}
		cur, err := protocol.DecodeWorksCursor(cursor)
		if err != nil {
			return nil, err
		}
		if cursor != "" && cur.UserID != userID {
			return nil, fmt.Errorf("the cursor belongs to another person")
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		tab := cur.Tab
		if tab == nil {
			body, err := protocol.UserWorksTabsBody(userID)
			if err != nil {
				return nil, err
			}
			raw, err := c.postUserProfile(ctx, p, userID, body)
			if err != nil {
				return nil, err
			}
			found, ok, err := protocol.ParseWorksTab(raw)
			if err != nil {
				return nil, err
			}
			if !ok {
				return &model.WorkPage{Items: []model.Work{}}, nil
			}
			tab = &found
		}
		body, err := protocol.UserWorksBody(userID, cur.Next, *tab)
		if err != nil {
			return nil, err
		}
		raw, err := c.postUserProfile(ctx, p, userID, body)
		if err != nil {
			return nil, err
		}
		res, err := protocol.ParseUserWorks(raw)
		if err != nil {
			return nil, err
		}
		page := &model.WorkPage{Items: res.Items, HasMore: res.HasMore}
		if res.HasMore && res.Cursor != "" && len(res.Items) > 0 {
			if page.Cursor, err = protocol.EncodeWorksCursor(protocol.WorksCursor{UserID: userID, Next: res.Cursor, Tab: tab}); err != nil {
				return nil, err
			}
		}
		return page, nil
	})
}

// postUserProfile 向某人主页的列表接口发送一个签过名的 body。
func (c *Client) postUserProfile(ctx context.Context, p *DeviceProfile, userID string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(protocol.PathUserWorks, userID)
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignPOST(path, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
	raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+path+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
	return raw, err
}
