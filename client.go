package ticktick

import (
	"context"
	"fmt"

	"github.com/carlmjohnson/requests"
	"github.com/tidwall/gjson"
)

const (
	BaseUrlV2       = "https://api.dida365.com/api/v2"
	SigninUrl       = BaseUrlV2 + "/user/signin"
	SettingUrl      = BaseUrlV2 + "/user/preferences/settings"
	InitialBatchUrl = BaseUrlV2 + "/batch/check/0"
)

type Client struct {
	UserName string
	PassWord string

	token           string
	inboxId         string
	projectGroups   []projectGroupsItem
	projectProfiles []projectProfilesItem
	tasks           []taskItem
	tags            []string
}

type projectGroupsItem struct {
	id   string
	name string
}

type projectProfilesItem struct {
	id   string
	name string
}

type taskItem struct {
	id        string
	projectId string
	title     string
	startDate string
	dueDate   string
	isAllDay  bool
	priority  int64
	tags      []string
}

// init the client struct (login, sync)
func (c *Client) Init() error {
	if err := c.GetToken(); err != nil {
		return err
	}
	if err := c.Sync(); err != nil {
		return err
	}
	return nil
}

// get the ticktick token
func (c *Client) GetToken() error {
	body := map[string]string{
		"username": c.UserName,
		"password": c.PassWord,
	}
	var resp string

	if err := requests.
		URL(SigninUrl).
		BodyJSON(&body).
		ToString(&resp).
		Fetch(context.Background()); err != nil {
		return err
	}

	token := gjson.Get(resp, "token").String()
	if token == "" {
		return fmt.Errorf("no token found in the response, full response json is %v", resp)
	}
	c.token = token
	return nil
}

func (c *Client) Sync() error {
	var resp string

	if err := requests.
		URL(InitialBatchUrl).
		Cookie("t", c.token).
		ToString(&resp).
		Fetch(context.Background()); err != nil {
		return err
	}

	// below we assume the apis are stable
	c.inboxId = gjson.Get(resp, "inboxId").String()
	gjson.Get(resp, "projectGroups").ForEach(func(key, value gjson.Result) bool {
		c.projectGroups = append(c.projectGroups, projectGroupsItem{
			id:   value.Get("id").String(),
			name: value.Get("name").String(),
		})
		return true
	})
	gjson.Get(resp, "projectProfiles").ForEach(func(key, value gjson.Result) bool {
		c.projectProfiles = append(c.projectProfiles, projectProfilesItem{
			id:   value.Get("id").String(),
			name: value.Get("name").String(),
		})
		return true
	})
	gjson.Get(resp, "syncTaskBean.update").ForEach(func(key, value gjson.Result) bool {
		var tags []string
		value.Get("tags").ForEach(func(_, tv gjson.Result) bool {
			tags = append(tags, tv.String())
			return true
		})

		if kind := value.Get("kind").String(); kind != "NOTE" {
			c.tasks = append(c.tasks, taskItem{
				id:        value.Get("id").String(),
				projectId: value.Get("projectId").String(),
				title:     value.Get("title").String(),
				startDate: value.Get("startDate").String(),
				dueDate:   value.Get("dueDate").String(),
				isAllDay:  value.Get("isAllDay").Bool(),
				priority:  value.Get("priority").Int(),
				tags:      tags,
			})
		}
		return true
	})
	gjson.Get(resp, "tags").ForEach(func(key, value gjson.Result) bool {
		c.tags = append(c.tags, value.Get("name").String())
		return true
	})

	// for _, v := range c.tasks {
	// 	fmt.Println(v)
	// }
	fmt.Println(c.tags)

	return nil
}
