package httpapi

import "task/internal/model"

type userResponse struct {
	User model.User `json:"user"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

type teamResponse struct {
	Team model.Team `json:"team"`
}

type teamsResponse struct {
	Teams []model.Team `json:"teams"`
}

type membersResponse struct {
	Members []model.TeamMember `json:"members"`
}

type taskResponse struct {
	Task model.Task `json:"task"`
}

type taskListResponse struct {
	Tasks  []model.Task `json:"tasks"`
	Total  int64        `json:"total"`
	Limit  int32        `json:"limit"`
	Offset int32        `json:"offset"`
}

type historyResponse struct {
	History []model.TaskHistory `json:"history"`
}

type commentResponse struct {
	Comment model.Comment `json:"comment"`
}

type commentsResponse struct {
	Comments []model.Comment `json:"comments"`
}

type statsResponse struct {
	Stats model.TeamStats `json:"stats"`
}

type statusResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error errorInfo `json:"error"`
}