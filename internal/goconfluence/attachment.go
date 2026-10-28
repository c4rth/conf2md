package goconfluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type Attachment struct {
	MediaTypeDescription string `json:"mediaTypeDescription"`
	WebuiLink            string `json:"webuiLink"`
	DownloadLink         string `json:"downloadLink"`
	CreatedAt            any    `json:"createdAt"`
	ID                   string `json:"id"`
	Comment              string `json:"comment"`
	Version              struct {
		Number    int       `json:"number"`
		Message   string    `json:"message"`
		MinorEdit bool      `json:"minorEdit"`
		AuthorID  string    `json:"authorId"`
		CreatedAt time.Time `json:"createdAt"`
	} `json:"version"`
	Title     string `json:"title"`
	FileSize  int    `json:"fileSize"`
	Status    string `json:"status"`
	PageID    string `json:"pageId"`
	FileID    string `json:"fileId"`
	MediaType string `json:"mediaType"`
	Links     struct {
		Download string `json:"download"`
		Webui    string `json:"webui"`
	} `json:"_links"`
}

// getAttachmentEndpoint creates the correct api endpoint by given id
func (a *API) getAttachmentEndpoint(id string) (*url.URL, error) {
	return a.getEndpoint("/attachments/" + id)
}

// GetAttachmentById Get info on specific attachment by id
func (a *API) GetAttachmentById(ctx context.Context, id string) (*Attachment, error) {
	ep, err := a.getAttachmentEndpoint(id)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var attachment Attachment

	err = json.Unmarshal(res, &attachment)
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// GetAttachmentByUrl Get attachment by download URL
func (a *API) GetAttachmentByUrl(ctx context.Context, downloadUrl string) (*[]byte, error) {
	ep, err := url.ParseRequestURI(a.BaseURL.String() + downloadUrl)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
