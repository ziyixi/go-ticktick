package ticktick

import (
	"fmt"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

// ********* test utils ********* //

func BuildSampleClient() *Client {
	NewSignInTestServer()
	syncResponse := BuildSyncResponse()
	NewSyncTestServer(syncResponse)
	client, _ := NewClient("testuser", "testpass", "test")

	return client
}

// ********* test part ********* //
func TestNewTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()

	// normal case
	exampleTime := time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC)
	task, err := NewTask(client, "test", "testcontent", exampleTime, "pname1")
	assert.Nil(err)
	assert.Equal(
		TaskItem{
			Title:       "test",
			Content:     "testcontent",
			StartDate:   exampleTime.Format(TemplateTime),
			ProjectId:   "pid1",
			ProjectName: "pname1",
		},
		*task,
	)

	// empty time
	task, err = NewTask(client, "test", "testcontent", time.Time{}, "pname1")
	assert.Nil(err)
	assert.Equal(
		TaskItem{
			Title:       "test",
			Content:     "testcontent",
			StartDate:   "",
			ProjectId:   "pid1",
			ProjectName: "pname1",
		},
		*task,
	)

	// project name not found
	task, err = NewTask(client, "test", "testcontent", time.Time{}, "randomProject")
	assert.Nil(task)
	if assert.NotNil(err) {
		assert.Contains(fmt.Sprint(err), "not found")
	}
}

func TestCreateTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()
	task, _ := NewTask(client, "test", "testcontent", time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC), "pname1")

	// test server
	taskWithId := *task
	taskWithId.Id = "testid"
	gock.New(baseUrlV2Test).
		Post(taskCreateUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		MatchType("json").
		JSON(task).
		Reply(200).
		JSON(&taskWithId)

	// normal case
	newtask, err := client.CreateTask(task)
	assert.Nil(err)
	assert.Equal(taskWithId, *newtask)

	// if the task already has an Id
	newtask, err = client.CreateTask(&taskWithId)
	assert.Nil(newtask)
	if assert.NotNil(err) {
		assert.Contains(fmt.Sprint(err), "the task has already been created with id")
	}

	// if no response due to the server error
	gock.New(baseUrlV2Test).
		Post(taskCreateUrlEndpoint).
		Reply(404)
	newtask, err = client.CreateTask(task)
	assert.Nil(newtask)
	assert.NotNil(err)
}

func TestDeleteTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()
	task, _ := NewTask(client, "test", "testcontent", time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC), "pname1")

	// test server
	type deleteElement struct {
		ProjectId string `json:"projectId"`
		TaskId    string `json:"taskId"`
	}
	body := struct {
		Delete []deleteElement `json:"delete"`
	}{
		Delete: []deleteElement{
			{
				ProjectId: "pid1",
				TaskId:    "1",
			},
		},
	}
	gock.New(baseUrlV2Test).
		Post(taskDeleteUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		MatchType("json").
		JSON(body).
		Reply(200)

	// normal case
	task.Id = "1" // mimic createTask
	ntask, err := client.DeleteTask(task)
	assert.NotNil(ntask)
	assert.Nil(err)
	assert.Empty(ntask.ProjectId)
	assert.Empty(ntask.ProjectName)
	assert.Empty(ntask.Id)

	// No Id
	task.Id = ""
	ntask, err = client.DeleteTask(task)
	assert.Nil(ntask)
	if assert.NotNil(err) {
		assert.Contains(fmt.Sprint(err), "the task has not been created, thus not deleted")
	}

	// server error
	task.Id = "1"
	gock.New(baseUrlV2Test).
		Post(taskDeleteUrlEndpoint).
		Reply(404)
	ntask, err = client.DeleteTask(task)
	assert.Nil(ntask)
	assert.NotNil(err)
}

func TestSearchTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()

	// normal case
	tasks, err := client.SearchTask("1", "pname1", "a", "", time.Time{}, time.Time{}, -1)
	assert.Nil(err)
	assert.True(len(tasks) == 1)
	tasks, err = client.SearchTask("", "", "", "", time.Time{}, time.Time{}, -1)
	assert.Nil(err)
	assert.True(len(tasks) > 1)

	// project not found
	tasks, err = client.SearchTask("1", "pnameRandom", "a", "", time.Time{}, time.Time{}, -1)
	assert.Nil(err)
	assert.True(len(tasks) == 0)

	// tags not found
	tasks, err = client.SearchTask("1", "pname1", "random", "", time.Time{}, time.Time{}, -1)
	assert.Nil(err)
	assert.True(len(tasks) == 0)

	// Id not found
	tasks, err = client.SearchTask("1", "pname1", "", "r", time.Time{}, time.Time{}, -1)
	assert.Nil(err)
	assert.True(len(tasks) == 0)

	// time not in the range
	tasks, err = client.SearchTask("1", "pname1", "", "", time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC), time.Date(2023, 01, 02, 11, 30, 59, 0, time.UTC), -1)
	assert.Nil(err)
	assert.True(len(tasks) == 0)

	// priority not correct
	tasks, err = client.SearchTask("1", "pname1", "", "", time.Time{}, time.Time{}, 3)
	assert.Nil(err)
	assert.True(len(tasks) == 0)
}

func TestUpdateTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()
	task, _ := NewTask(client, "test", "testcontent", time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC), "pname1")
	task.Id = "1" // mimic createTask

	// test server
	taskUpdated := *task
	taskUpdated.Priority = 5
	gock.New(baseUrlV2Test).
		Post(fmt.Sprintf(taskUpdateUrlEndpoint, task.Id)).
		MatchHeader("Cookie", "t=testtoken").
		MatchType("json").
		JSON(taskUpdated).
		Reply(200).
		JSON(&taskUpdated)

	// normal case
	ntask, err := client.UpdateTask(&taskUpdated)
	assert.Equal(&taskUpdated, ntask)
	assert.Nil(err)

	// Empty Id
	taskUpdated.Id = ""
	ntask, err = client.UpdateTask(&taskUpdated)
	assert.Nil(ntask)
	if assert.NotNil(err) {
		assert.Contains(fmt.Sprint(err), "task Id is empty")
	}

	// server error
	taskUpdated.Id = "1"
	gock.New(baseUrlV2Test).
		Post(fmt.Sprintf(taskUpdateUrlEndpoint, task.Id)).
		Reply(404)
	ntask, err = client.UpdateTask(&taskUpdated)
	assert.Nil(ntask)
	assert.NotNil(err)
}

func TestCompleteTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()
	task, _ := NewTask(client, "test", "testcontent", time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC), "pname1")
	task.Id = "1" // mimic createTask

	// test server
	gock.New(baseUrlV2Test).
		Post(fmt.Sprintf(taskUpdateUrlEndpoint, task.Id)).
		MatchHeader("Cookie", "t=testtoken").
		Reply(200).
		JSON(task)

	// normal case
	ntask, err := client.CompleteTask(task)
	assert.NotNil(ntask)
	assert.Nil(err)
}

func TestMakeSubtask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()

	// test server: subtask
	type bodyElement struct {
		ParentId  string `json:"parentId"`
		ProjectId string `json:"projectId"`
		TaskId    string `json:"taskId"`
	}
	var body []bodyElement
	body = append(body, bodyElement{
		ParentId:  "p",
		ProjectId: "pid1",
		TaskId:    "c",
	})
	gock.New(baseUrlV2Test).
		Post(MakeSubtaskUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		MatchType("json").
		JSON(body).
		Reply(200)

	// normal case, when both are in the same dir
	taskp, _ := client.SearchTask("1", "", "", "", time.Time{}, time.Time{}, -1)
	taskc, _ := client.SearchTask("2", "", "", "", time.Time{}, time.Time{}, -1)
	newp, newt, err := client.MakeSubtask(&taskp[0], &taskc[0])
	assert.NotNil(newp)
	assert.NotNil(newt)
	assert.Nil(err)

	// when taskp has no Id
	taskp[0].Id = ""
	_, _, err = client.MakeSubtask(&taskp[0], &taskc[0])
	if assert.NotNil(err) {
		assert.Contains(fmt.Sprint(err), "the parent has not been created")
	}
	taskp[0].Id = "1"

	// when taskc has no Id
	taskc[0].Id = ""
	_, _, err = client.MakeSubtask(&taskp[0], &taskc[0])
	if assert.NotNil(err) {
		assert.Contains(fmt.Sprint(err), "the child has not been created")
	}
	taskc[0].Id = "1"

	// when we have to move project
	// test server: move project
	taskc, _ = client.SearchTask("3", "", "", "", time.Time{}, time.Time{}, -1)
	taskcUpdated := taskc[0]
	taskcUpdated.ProjectId = "pid1"
	gock.New(baseUrlV2Test).
		Post(MoveTaskUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		Reply(200).
		JSON(taskcUpdated)
	gock.New(baseUrlV2Test).
		Post(MakeSubtaskUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		Reply(200)
	newp, newt, err = client.MakeSubtask(&taskp[0], &taskc[0])
	assert.NotNil(newp)
	assert.NotNil(newt)
	assert.Nil(err)

	// if server error
	gock.New(baseUrlV2Test).
		Post(MakeSubtaskUrlEndpoint).
		Reply(404)
	taskp, _ = client.SearchTask("1", "", "", "", time.Time{}, time.Time{}, -1)
	taskc, _ = client.SearchTask("2", "", "", "", time.Time{}, time.Time{}, -1)
	newp, newt, err = client.MakeSubtask(&taskp[0], &taskc[0])
	assert.Nil(newp)
	assert.Nil(newt)
	assert.NotNil(err)
}

func TestMoveTask(t *testing.T) {
	defer gock.Off()
	assert := assert.New(t)
	client := BuildSampleClient()

	task, _ := NewTask(client, "test", "testcontent", time.Date(2023, 01, 01, 11, 30, 59, 0, time.UTC), "pname1")
	task.Id = "test"

	// test server
	type bodyElement struct {
		FromProjectId string `json:"fromProjectId"`
		TaskId        string `json:"taskId"`
		ToProjectId   string `json:"toProjectId"`
	}
	var body []bodyElement
	body = append(body, bodyElement{
		FromProjectId: "pid1",
		TaskId:        "test",
		ToProjectId:   "pid2",
	})
	gock.New(baseUrlV2Test).
		Post(MoveTaskUrlEndpoint).
		MatchHeader("Cookie", "t=testtoken").
		MatchType("json").
		JSON(body).
		Reply(200)

	// normal case
	newt, err := client.MoveTask(task, "pname2")
	assert.NotNil(newt)
	assert.Nil(err)

	// if move to the same project
	newt, err = client.MoveTask(task, "pname1")
	assert.NotNil(newt)
	assert.Nil(err)
	assert.Equal(newt, task)

	// if the new project not exists
	newt, err = client.MoveTask(task, "pnameRandom")
	assert.Nil(newt)
	assert.NotNil(err)

	// if server error
	gock.New(baseUrlV2Test).
		Post(MoveTaskUrlEndpoint).
		Reply(404)
	newt, err = client.MoveTask(task, "pname2")
	assert.Nil(newt)
	assert.NotNil(err)
}
