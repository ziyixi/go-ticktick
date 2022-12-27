package ticktick

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/carlmjohnson/requests"
)

const (
	taskCreateUrl   = baseUrlV2 + "/task"
	taskDeleteUrl   = baseUrlV2 + "/batch/task"
	taskUpdateUrl   = baseUrlV2 + "/task/%v"
	taskCompleteUrl = baseUrlV2 + "/project/%v/task/%v/complete"
)

type TaskItem struct {
	Id          string `json:"id"`
	ProjectId   string `json:"projectId"`
	ProjectName string `json:"-"`

	Title string `json:"title"`

	IsAllDay  bool       `json:"isAllDay"`
	Tags      []string   `json:"tags"`
	Content   string     `json:"content"`
	Desc      string     `json:"desc"`
	AllDay    bool       `json:"allDay"`
	StartDate string     `json:"startDate"` // the dates are all in UTC
	DueDate   string     `json:"dueDate"`   // and will not be influenced by TimeZone
	TimeZone  string     `json:"timeZone"`
	Reminders []string   `json:"reminders"`
	Repeat    string     `json:"repeat"`
	Priority  int64      `json:"priority"`
	SortOrder int64      `json:"sortOrder"`
	Kind      string     `json:"kind"`
	Items     []TaskItem `json:"Items"`
}

func NewTask(c *Client, title string, content string, startDate time.Time, projectName string) (*TaskItem, error) {
	projectId := ""
	if projectName != "" {
		pid, ok := c.project2Id[projectName]
		if !ok {
			return nil, fmt.Errorf("projectName %v not found", projectName)
		} else {
			projectId = pid
		}
	}
	t := TaskItem{
		Title:       title,
		Content:     content,
		StartDate:   startDate.Format(TemplateTime),
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
		URL(taskCreateUrl).
		Cookie("t", c.loginToken).
		BodyJSON(t).
		ToJSON(&resp).
		Fetch(context.Background()); err != nil {
		return nil, err
	}

	resp.ProjectName = c.id2Project[resp.ProjectId]
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

	if body.Delete[0].ProjectId == "inbox" {
		body.Delete[0].ProjectId = c.inboxId
	}

	if err := requests.
		URL(taskDeleteUrl).
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
func (c *Client) SearchTask(title string, project string, tag string, StartDateNotbefore time.Time, StartDateNotafter time.Time, priority int64) ([]TaskItem, error) {
	c.Sync()

	var res []TaskItem
	for _, task := range c.tasks {
		if !(strings.Contains(task.Title, title)) {
			continue
		}
		if expectPId, ok := c.project2Id[project]; (project != "") && (!ok || expectPId != task.ProjectId) {
			continue
		}
		if tag != "" && !Contains(task.Tags, tag) {
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
		URL(fmt.Sprintf(taskUpdateUrl, t.Id)).
		Cookie("t", c.loginToken).
		BodyJSON(t).
		ToJSON(&resp).
		Fetch(context.Background()); err != nil {
		return nil, err
	}

	resp.ProjectName = c.id2Project[resp.ProjectId]
	return &resp, nil
}
