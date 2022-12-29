package ticktick

import (
	"fmt"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

// ********* test utils ********* //

type TestProjectItem struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type TestTagItem struct {
	Name string `json:"name"`
}

type testsyncTask struct {
	Update []TaskItem `json:"update"`
}

type TestSyncResponse struct {
	InboxId         string            `json:"inboxId"`
	ProjectGroups   []TestProjectItem `json:"projectGroups"`
	ProjectProfiles []TestProjectItem `json:"projectProfiles"`
	SyncTaskBean    testsyncTask      `json:"syncTaskBean"`
	Tags            []TestTagItem     `json:"tags"`
}

func BuildSyncResponse() *TestSyncResponse {
	sampleResponse := TestSyncResponse{
		InboxId: "testinboxid",
		ProjectGroups: []TestProjectItem{
			{
				Name: "pgname1",
				Id:   "pgid1",
			},
			{
				Name: "pgname2",
				Id:   "pgid2",
			},
		},
		ProjectProfiles: []TestProjectItem{
			{
				Name: "pname1",
				Id:   "pid1",
			},
			{
				Name: "pname2",
				Id:   "pid2",
			},
		},
		SyncTaskBean: testsyncTask{
			Update: []TaskItem{
				{
					Id:        "1",
					Title:     "1",
					ProjectId: "pid1",
					Tags:      []string{"a", "b"},
					StartDate: "2022-12-12T15:04:05.000+0000",
					Priority:  5,
				},
				{
					Id:        "2",
					Title:     "2",
					ProjectId: "pid1",
					Tags:      []string{"b", "c"},
					StartDate: "2022-12-13T15:04:05.000+0000",
					Priority:  0,
				},
				{
					Id:        "3",
					Title:     "3",
					ProjectId: "pid2",
					Tags:      []string{"b", "c"},
					StartDate: "2022-12-14T15:04:05.000+0000",
					Priority:  1,
				},
			},
		},
		Tags: []TestTagItem{
			{
				Name: "a",
			},
			{
				Name: "b",
			},
			{
				Name: "c",
			},
		},
	}
	return &sampleResponse
}

func NewSignInTestServer() {
	gock.New(baseUrlV2Test).
		Post(signinUrlEndpoint).
		MatchType("json").
		JSON(map[string]string{
			"username": "testuser",
			"password": "testpass",
		}).
		Reply(200).
		JSON(map[string]string{"token": "testtoken"})
}

func NewSyncTestServer(syncResponse *TestSyncResponse) {
	gock.New(baseUrlV2Test).
		Get(queryUnfinishedJobUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		Reply(200).
		JSON(syncResponse)
}

func NewSignInTestServerWithProblem() {
	gock.New(baseUrlV2Test).
		Post(signinUrlEndpoint).
		MatchType("json").
		JSON(map[string]string{
			"username": "testuser",
			"password": "testpass",
		}).
		Reply(200).
		JSON(map[string]string{"faketoken": "testtoken"})
}

// ********* test part ********* //

func TestGetToken(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)

	// normal case
	NewSignInTestServer()
	client := &Client{
		UserName:       "testuser",
		PassWord:       "testpass",
		baseUrlV2:      baseUrlV2Test,
		projectName2Id: make(map[string]string),
		id2ProjectName: make(map[string]string),
	}
	err := client.GetToken()
	assert.Nil(err)
	assert.Equal("testtoken", client.loginToken)

	// if no token is returned
	NewSignInTestServerWithProblem()
	err = client.GetToken()
	if assert.NotNil(t, err) {
		assert.Contains(fmt.Sprint(err), "no token found in the response, full response json is")
	}

	// if the server responses 404
	gock.New(baseUrlV2Test).
		Post(signinUrlEndpoint).
		Reply(404)
	err = client.GetToken()
	assert.NotNil(err)
}

func TestSync(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)

	// build response
	syncResponse := BuildSyncResponse()
	NewSyncTestServer(syncResponse)

	// normal case
	client := &Client{
		UserName:       "testuser",
		PassWord:       "testpass",
		baseUrlV2:      baseUrlV2Test,
		projectName2Id: make(map[string]string),
		id2ProjectName: make(map[string]string),
		loginToken:     "testtoken",
	}
	err := client.Sync()
	assert.Nil(err)

	// if the server responses 404
	gock.New(baseUrlV2Test).
		Get(queryUnfinishedJobUrlEndpoint).
		Reply(404)
	err = client.Sync()
	assert.NotNil(t, err)
}

func TestInit(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)

	NewSignInTestServer()
	syncResponse := BuildSyncResponse()
	NewSyncTestServer(syncResponse)

	// normal case
	client := &Client{
		UserName:       "testuser",
		PassWord:       "testpass",
		baseUrlV2:      baseUrlV2Test,
		projectName2Id: make(map[string]string),
		id2ProjectName: make(map[string]string),
	}
	err := client.Init()
	assert.Nil(err)

	// if GetToken has problem
	gock.New(baseUrlV2Test).
		Post(signinUrlEndpoint).
		Reply(404)
	err = client.Init()
	assert.NotNil(err)

	// if sync has problem
	NewSignInTestServer()
	gock.New(baseUrlV2Test).
		Get(queryUnfinishedJobUrlEndpoint).
		Reply(404)

	err = client.Init()
	assert.NotNil(err)
}

func TestNewClient(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)

	// normal case
	NewSignInTestServer()
	syncResponse := BuildSyncResponse()
	NewSyncTestServer(syncResponse)

	client, err := NewClient("testuser", "testpass", "test")
	assert.Nil(err)
	assert.NotNil(client)

	// if the server name is not correct
	client, err = NewClient("testuser", "testpass", "random")
	assert.NotNil(err)
	assert.Nil(client)

	// if Init has problem
	gock.New(baseUrlV2Test).
		Post(signinUrlEndpoint).
		Reply(404)
	client, err = NewClient("testuser", "testpass", "test")
	assert.NotNil(err)
	assert.Nil(client)

}
