package models

type Result struct {
	DeletedResources []Resource `json:"deleted_resources"`
	FailedResources  []Resource `json:"failed_resources"`
	PendingResources []Resource `json:"pending_resources,omitempty"`
}
