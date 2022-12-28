package ticktick

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlmjohnson/requests"
	"github.com/tidwall/gjson"
)

const (
	baseUrlV2             = "https://api.dida365.com/api/v2"
	signinUrl             = baseUrlV2 + "/user/signin"               // POST
	settingUrl            = baseUrlV2 + "/user/preferences/settings" // POST
	queryUnfinishedJobUrl = baseUrlV2 + "/batch/check/0"             // GET
)

type Client struct {
	UserName string
	PassWord string

	loginToken string
	inboxId    string

	projectGroups []projectGroupsItem

	projectName2Id map[string]string
	id2ProjectName map[string]string

	tasks []TaskItem

	tags []string
}

type projectGroupsItem struct {
	id   string
	name string
}

func NewClient(userName, passWord string) (*Client, error) {
	client := &Client{UserName: userName, PassWord: passWord, projectName2Id: make(map[string]string), id2ProjectName: make(map[string]string)}
	if err := client.Init(); err != nil {
		return nil, err
	}
	return client, nil
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
		URL(signinUrl).
		BodyJSON(&body).
		ToString(&resp).
		Fetch(context.Background()); err != nil {
		return err
	}

	loginToken := gjson.Get(resp, "token").String()
	if loginToken == "" {
		return fmt.Errorf("no token found in the response, full response json is %v", resp)
	}
	c.loginToken = loginToken
	return nil
}

// fetch all the user contents
func (c *Client) Sync() error {
	var resp string

	if err := requests.
		URL(queryUnfinishedJobUrl).
		Cookie("t", c.loginToken).
		ToString(&resp).
		Fetch(context.Background()); err != nil {
		return err
	}

	// below we assume the apis are stable
	c.inboxId = gjson.Get(resp, "inboxId").String()

	c.projectGroups = nil
	gjson.Get(resp, "projectGroups").ForEach(func(key, value gjson.Result) bool {
		c.projectGroups = append(c.projectGroups, projectGroupsItem{
			id:   value.Get("id").String(),
			name: value.Get("name").String(),
		})
		return true
	})

	c.projectName2Id = make(map[string]string)
	c.projectName2Id["inbox"] = c.inboxId
	c.id2ProjectName = make(map[string]string)
	c.id2ProjectName[c.inboxId] = "inbox"
	gjson.Get(resp, "projectProfiles").ForEach(func(key, value gjson.Result) bool {
		c.projectName2Id[value.Get("name").String()] = value.Get("id").String()
		c.id2ProjectName[value.Get("id").String()] = value.Get("name").String()
		return true
	})

	c.tasks = nil
	gjson.Get(resp, "syncTaskBean.update").ForEach(func(key, value gjson.Result) bool {
		var t TaskItem
		if err := json.Unmarshal([]byte(value.Raw), &t); err != nil {
			panic("task Unmarshal failed for " + value.Raw)
		}
		t.ProjectName = c.id2ProjectName[t.ProjectId]
		c.tasks = append(c.tasks, t)
		return true
	})

	c.tags = nil
	gjson.Get(resp, "tags").ForEach(func(key, value gjson.Result) bool {
		c.tags = append(c.tags, value.Get("name").String())
		return true
	})

	return nil
}
