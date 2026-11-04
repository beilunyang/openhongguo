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

// PersonWorks 加载一位演员参演的短剧，一次一页（主页上的作品 › 剧作）。
// cursor 是上一页的 Cursor。
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
		body, err := protocol.UserWorksBody(userID, cur.Next)
		if err != nil {
			return nil, err
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		path := fmt.Sprintf(protocol.PathUserWorks, userID)
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignPOST(path, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
		raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+path+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
		if err != nil {
			return nil, err
		}
		res, err := protocol.ParseUserWorks(raw)
		if err != nil {
			return nil, err
		}
		page := &model.WorkPage{Items: res.Items, HasMore: res.HasMore}
		if res.HasMore && res.Cursor != "" && len(res.Items) > 0 {
			if page.Cursor, err = protocol.EncodeWorksCursor(protocol.WorksCursor{UserID: userID, Next: res.Cursor}); err != nil {
				return nil, err
			}
		}
		return page, nil
	})
}
