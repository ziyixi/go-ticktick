package ticktick

import (
	"context"
	"fmt"
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
	Id        string `json:"id"`
	ProjectId string `json:"projectId"`

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
		pid, ok := c.projectProfiles[projectName]
		if !ok {
			return nil, fmt.Errorf("projectName %v not found", projectName)
		} else {
			projectId = pid
		}
	}
	t := TaskItem{
		Title:     title,
		Content:   content,
		StartDate: startDate.Format(time.RFC3339Nano),
		ProjectId: projectId,
	}
	return &t, nil
}

func (t *TaskItem) Create(c *Client) (*TaskItem, error) {
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

	return &resp, nil
}

func (t *TaskItem) Delete(c *Client) (*TaskItem, error) {
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
	newt.Id = ""

	return &newt, nil
}
