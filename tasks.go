package ticktick

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/carlmjohnson/requests"
)

const (
	taskCreateUrlEndpoint  = "/task"              // POST
	taskDeleteUrlEndpoint  = "/batch/task"        // POST
	taskUpdateUrlEndpoint  = "/task/%v"           // POST
	MakeSubtaskUrlEndpoint = "/batch/taskParent"  // POST
	MoveTaskUrlEndpoint    = "/batch/taskProject" // POST
)

type TaskItem struct {
	Id          string `json:"id"`
	ProjectId   string `json:"projectId"`
	ProjectName string `json:"-"`
	ParentId    string `json:"parentId"`

	Title string `json:"title"`

	IsAllDay  bool     `json:"isAllDay"`
	Tags      []string `json:"tags"`
	Content   string   `json:"content"`
	Desc      string   `json:"desc"`
	AllDay    bool     `json:"allDay"`
	StartDate string   `json:"startDate"` // the dates are all in UTC
	DueDate   string   `json:"dueDate"`   // and will not be influenced by TimeZone
	TimeZone  string   `json:"timeZone"`
	Reminders []string `json:"reminders"`
	Repeat    string   `json:"repeat"`
	Priority  int64    `json:"priority"`
	SortOrder int64    `json:"sortOrder"`
	Kind      string   `json:"kind"`
	Status    int64    `json:"status"`
}

func NewTask(c *Client, title string, content string, startDate time.Time, projectName string) (*TaskItem, error) {
	projectId := ""
	if projectName != "" {
		pid, ok := c.projectName2Id[projectName]
		if !ok {
			return nil, fmt.Errorf("projectName %v not found", projectName)
		} else {
			projectId = pid
		}
	}

	startDateString := ""
	if !startDate.IsZero() {
		startDateString = startDate.Format(TemplateTime)
	}

	t := TaskItem{
		Title:       title,
		Content:     content,
		StartDate:   startDateString,
		ProjectId:   projectId,
		ProjectName: projectName,
	}
	return &t, nil
}

// CURD, Create
func (c *Client) CreateTask(t *TaskItem) (*TaskItem, error) {
	if t.Id != "" {
		return nil, fmt.Errorf("the task has already been created with id=%v", t.Id)
	}
	var resp TaskItem
	if err := requests.
		URL(c.baseUrlV2+taskCreateUrlEndpoint).
		Cookie("t", c.loginToken).
		BodyJSON(t).
		ToJSON(&resp).
		Fetch(context.Background()); err != nil {
		return nil, err
	}

	resp.ProjectName = c.id2ProjectName[resp.ProjectId]
	return &resp, nil
}

// CURD, Delete
func (c *Client) DeleteTask(t *TaskItem) (*TaskItem, error) {
	if t.Id == "" {
		return nil, fmt.Errorf("the task has not been created, thus not deleted")
	}

	type deleteElement struct {
		ProjectId string `json:"projectId"`
		TaskId    string `json:"taskId"`
	}
	body := struct {
		Delete []deleteElement `json:"delete"`
	}{
		Delete: []deleteElement{
			{
				ProjectId: t.ProjectId,
				TaskId:    t.Id,
			},
		},
	}

	if err := requests.
		URL(c.baseUrlV2+taskDeleteUrlEndpoint).
		Cookie("t", c.loginToken).
		BodyJSON(&body).
		Fetch(context.Background()); err != nil {
		return nil, err
	}

	// reset task
	newt := *t
	newt.ProjectId = ""
	newt.ProjectName = ""
	newt.Id = ""

	return &newt, nil
}

// CURD, Read, partial match. if parameter is "", it will have no effect.
// If priority is -1, it's ignored. If time is zero val, it's ignored.
// Priority values are: 0,1,3,5 (low -> high)
func (c *Client) SearchTask(title string, project string, tag string, id string, StartDateNotbefore time.Time, StartDateNotafter time.Time, priority int64) ([]TaskItem, error) {
	c.Sync()

	var res []TaskItem
	for _, task := range c.tasks {
		if !(strings.Contains(task.Title, title)) {
			continue
		}
		if expectPId, ok := c.projectName2Id[project]; (project != "") && (!ok || expectPId != task.ProjectId) {
			continue
		}
		if tag != "" && !Contains(task.Tags, tag) {
			continue
		}
		if id != "" && task.Id != id {
			continue
		}
		if taskTime, _ := time.Parse(TemplateTime, task.StartDate); !StartDateNotbefore.IsZero() && !StartDateNotafter.IsZero() && (taskTime.Before(StartDateNotbefore) || taskTime.After(StartDateNotafter)) {
			continue
		}
		if priority != -1 && priority != task.Priority {
			continue
		}
		res = append(res, task)
	}

	return res, nil
}

// CURD, Update
func (c *Client) UpdateTask(t *TaskItem) (*TaskItem, error) {
	if t.Id == "" {
		return nil, fmt.Errorf("task Id is empty")
	}
	var resp TaskItem
	if err := requests.
		URL(fmt.Sprintf(c.baseUrlV2+taskUpdateUrlEndpoint, t.Id)).
		Cookie("t", c.loginToken).
		BodyJSON(t).
		ToJSON(&resp).
		Fetch(context.Background()); err != nil {
		return nil, err
	}

	resp.ProjectName = c.id2ProjectName[resp.ProjectId]
	return &resp, nil
}

// Complete task, as complete has no field
func (c *Client) CompleteTask(t *TaskItem) (*TaskItem, error) {
	newt := *t
	newt.Status = 2
	return c.UpdateTask(&newt)
}

// Make subtask, p is the parent, t is the child, return the parent and child tasks
func (c *Client) MakeSubtask(p, t *TaskItem) (*TaskItem, *TaskItem, error) {
	if p.Id == "" {
		return nil, nil, fmt.Errorf("the parent has not been created")
	}
	if t.Id == "" {
		return nil, nil, fmt.Errorf("the child has not been created")
	}

	// if p and t not in the same project, move t to p's project
	if p.ProjectId != t.ProjectId {
		newt, err := c.MoveTask(t, p.ProjectName)
		if err != nil {
			return nil, nil, err
		}
		t = newt
	}

	type bodyElement struct {
		ParentId  string `json:"parentId"`
		ProjectId string `json:"projectId"`
		TaskId    string `json:"taskId"`
	}
	var body []bodyElement
	body = append(body, bodyElement{
		ParentId:  p.Id,
		ProjectId: p.ProjectId,
		TaskId:    t.Id,
	})

	if err := requests.
		URL(c.baseUrlV2+MakeSubtaskUrlEndpoint).
		Cookie("t", c.loginToken).
		BodyJSON(body).
		Fetch(context.Background()); err != nil {
		return nil, nil, err
	}

	// as the response is not the task itself, we sync and search
	c.Sync()
	newPList, err := c.SearchTask("", "", "", p.Id, time.Time{}, time.Time{}, -1)
	if err != nil {
		return nil, nil, err
	}
	if len(newPList) == 0 {
		return nil, nil, fmt.Errorf("server error in response to get the parent task")
	}
	newCList, err := c.SearchTask("", "", "", t.Id, time.Time{}, time.Time{}, -1)
	if err != nil {
		return nil, nil, err
	}
	if len(newCList) == 0 {
		return nil, nil, fmt.Errorf("server error in response to get the child task")
	}
	return &newPList[0], &newCList[0], nil
}

// Move task to another project, as directly updating projectId has no effect
func (c *Client) MoveTask(t *TaskItem, to string) (*TaskItem, error) {
	if t.ProjectName == to {
		return t, nil
	}
	toId, ok := c.projectName2Id[to]
	if !ok {
		return nil, fmt.Errorf("the project name %v not exist", to)
	}

	type bodyElement struct {
		FromProjectId string `json:"fromProjectId"`
		TaskId        string `json:"taskId"`
		ToProjectId   string `json:"toProjectId"`
	}
	var body []bodyElement
	body = append(body, bodyElement{
		FromProjectId: t.ProjectId,
		TaskId:        t.Id,
		ToProjectId:   toId,
	})

	if err := requests.
		URL(c.baseUrlV2+MoveTaskUrlEndpoint).
		Cookie("t", c.loginToken).
		BodyJSON(body).
		Fetch(context.Background()); err != nil {
		return nil, err
	}

	newt := *t
	newt.ProjectId = toId
	return &newt, nil
}
