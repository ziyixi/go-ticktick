package ticktick

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlmjohnson/requests"
	"github.com/tidwall/gjson"
)

const (
	baseUrlV2Dida365              = "https://api.dida365.com/api/v2"
	baseUrlV2Ticktick             = "https://api.ticktick.com/api/v2"
	baseUrlV2Test                 = "https://api.test.com/api/v2"
	signinUrlEndpoint             = "/user/signin"   // POST
	queryUnfinishedJobUrlEndpoint = "/batch/check/0" // GET
)

type Client struct {
	UserName  string
	PassWord  string
	baseUrlV2 string

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

// create a new client, the server can be ticktick, dida365, test
func NewClient(userName, passWord, server string) (*Client, error) {
	var baseUrlV2 string
	switch server {
	case "ticktick":
		baseUrlV2 = baseUrlV2Ticktick
	case "dida365":
		baseUrlV2 = baseUrlV2Dida365
	case "test":
		baseUrlV2 = baseUrlV2Test
	default:
		return nil, fmt.Errorf("server name %v is not supported", server)
	}

	client := &Client{UserName: userName, PassWord: passWord, baseUrlV2: baseUrlV2, projectName2Id: make(map[string]string), id2ProjectName: make(map[string]string)}
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
		URL(c.baseUrlV2 + signinUrlEndpoint).
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
		URL(c.baseUrlV2+queryUnfinishedJobUrlEndpoint).
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
		json.Unmarshal([]byte(value.Raw), &t)
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
