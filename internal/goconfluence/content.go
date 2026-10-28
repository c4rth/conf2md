package goconfluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Results array
type Results struct {
	ID       string    `json:"id,omitempty"`
	Type     string    `json:"type,omitempty"`
	Status   string    `json:"status,omitempty"`
	Content  Content   `json:"content"`
	Excerpt  string    `json:"excerpt,omitempty"`
	Title    string    `json:"title,omitempty"`
	URL      string    `json:"url,omitempty"`
	Links    *Links    `json:"_links,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

// Content specifies content properties
type Content struct {
	ID        string     `json:"id,omitempty"`
	Type      string     `json:"type"`
	Status    string     `json:"status,omitempty"`
	Title     string     `json:"title"`
	Ancestors []Ancestor `json:"ancestors,omitempty"`
	Body      Body       `json:"body"`
	Version   *Version   `json:"version,omitempty"`
	Space     *Space     `json:"space"`
	History   *History   `json:"history,omitempty"`
	Links     *Links     `json:"_links,omitempty"`
	Metadata  *Metadata  `json:"metadata,omitempty"`
	Children  *Children  `json:"children,omitempty"`
}

type Children struct {
	Page *Search `json:"page,omitempty"`
}

// Metadata specifies metadata properties
type Metadata struct {
	Properties *Properties `json:"properties"`
	MediaType  string      `json:"mediaType,omitempty"`
	Comment    string      `json:"comment,omitempty"`
	FileSize   int         `json:"fileSize,omitempty"`
	FileId     string      `json:"fileId,omitempty"`
}

// Properties defines properties of the editor
type Properties struct {
	Editor                     *Editor                     `json:"editor,omitempty"`
	ContentAppearanceDraft     *ContentAppearanceDraft     `json:"content-appearance-draft"`
	ContentAppearancePublished *ContentAppearancePublished `json:"content-appearance-published"`
}

// Editor contains editor information
type Editor struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ContentAppearanceDraft sets the appearance of the content in draft form
type ContentAppearanceDraft struct {
	Value string `json:"value"`
}

// ContentAppearancePublished sets the appearance of the content in published form
type ContentAppearancePublished struct {
	Value string `json:"value"`
}

// Links contains link information
type Links struct {
	Base     string `json:"base"`
	TinyUI   string `json:"tinyui"`
	WebUI    string `json:"webui"`
	Download string `json:"download"`
	Self     string `json:"self"`
}

// Ancestor defines ancestors to create sub pages
type Ancestor struct {
	ID string `json:"id"`
}

// Body holds the storage information
type Body struct {
	Storage Storage  `json:"storage"`
	View    *Storage `json:"view,omitempty"`
}

// BodyExportView holds the export_view information
type BodyExportView struct {
	ExportView *Storage `json:"export_view"`
	View       *Storage `json:"view,omitempty"`
}

// Storage defines the storage information
type Storage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Version defines the content version number
// the version number is used for updating content
type Version struct {
	Number    int    `json:"number"`
	MinorEdit bool   `json:"minorEdit"`
	Message   string `json:"message,omitempty"`
	By        *User  `json:"by,omitempty"`
	When      string `json:"when,omitempty"`
}

// Space holds the Space information of a Content Page
type Space struct {
	ID     int    `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
}

// getContentIDEndpoint creates the correct api endpoint by given id
func (a *API) getContentIDEndpoint(id string) (*url.URL, error) {
	return a.getEndpoint("/content/" + id)
}

// getContentEndpoint creates the correct api endpoint
func (a *API) getContentEndpoint() (*url.URL, error) {
	return a.getEndpoint("/content/")
}

// getContentChildEndpoint creates the correct api endpoint by given id and type
func (a *API) getContentChildEndpoint(id string, t string) (*url.URL, error) {
	return a.getEndpoint("/content/" + id + "/child/" + t)
}

// getContentGenericEndpoint creates the correct api endpoint by given id and type
func (a *API) getContentGenericEndpoint(id string, t string) (*url.URL, error) {
	return a.getEndpoint("/content/" + id + "/" + t)
}

// ContentQuery defines the query parameters
// used for content related searching
// Query parameter values https://developer.atlassian.com/cloud/confluence/rest/#api-content-get
type ContentQuery struct {
	Expand     []string
	Limit      int    // page limit
	OrderBy    string // fieldpath asc/desc e.g: "history.createdDate desc"
	PostingDay string // required for blogpost type Format: yyyy-mm-dd
	SpaceKey   string
	Start      int    // page start
	Status     string // current, trashed, draft, any
	Title      string // required for page
	Trigger    string // viewed
	Type       string // page, blogpost
	Version    int    //version number when not lastest

}

// GetContentByID querys content by id
func (a *API) GetContentByID(ctx context.Context, id string, query ContentQuery) (*Content, error) {
	ep, err := a.getContentIDEndpoint(id)
	if err != nil {
		return nil, err
	}
	ep.RawQuery = addContentQueryParams(query).Encode()
	return a.SendContentRequest(ctx, ep, "GET", nil)
}

// ContentSearch results
type ContentSearch struct {
	Results []Content `json:"results"`
	Start   int       `json:"start,omitempty"`
	Limit   int       `json:"limit,omitempty"`
	Size    int       `json:"size,omitempty"`
}

// GetContent querys content using a query parameters
func (a *API) GetContent(ctx context.Context, query ContentQuery) (*ContentSearch, error) {
	ep, err := a.getContentEndpoint()
	if err != nil {
		return nil, err
	}
	ep.RawQuery = addContentQueryParams(query).Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var search ContentSearch

	err = json.Unmarshal(res, &search)
	if err != nil {
		return nil, err
	}
	return &search, nil
}

// GetChildPages returns a content list of child page objects
func (a *API) GetChildPages(ctx context.Context, id string) (*Search, error) {
	return a.getChildContent(ctx, id, "page")
}

func (a *API) getChildContent(ctx context.Context, id, contentType string) (*Search, error) {
	var (
		results      []Results
		searchResult Search
	)

	ep, err := a.getContentChildEndpoint(id, contentType)
	if err != nil {
		return nil, err
	}

	query := ContentQuery{
		Start: 0,
		Limit: 25,
	}

	searchResult.Start = 0

	for {
		ep.RawQuery = addContentQueryParams(query).Encode()
		s, err := a.SendSearchRequest(ctx, ep, "GET")
		if err != nil {
			return nil, err
		}
		results = append(results, s.Results...)
		pageLimit := query.Limit
		if s.Limit > 0 && s.Limit < pageLimit {
			pageLimit = s.Limit
		}
		if len(s.Results) < pageLimit {
			break
		}
		query.Start += len(s.Results)
	}

	searchResult.Limit = len(results)
	searchResult.Size = len(results)
	searchResult.Results = results

	return &searchResult, nil
}

// GetComments returns a list of comments belonging to id
func (a *API) GetComments(ctx context.Context, id string) (*Search, error) {
	ep, err := a.getContentChildEndpoint(id, "comment")
	if err != nil {
		return nil, err
	}
	return a.SendSearchRequest(ctx, ep, "GET")
}

// GetAttachments returns all attachments belonging to id.
func (a *API) GetAttachments(ctx context.Context, id string) (*Search, error) {
	return a.getChildContent(ctx, id, "attachment")
}

// History contains object history information
type History struct {
	LastUpdated LastUpdated `json:"lastUpdated"`
	Latest      bool        `json:"latest"`
	CreatedBy   User        `json:"createdBy"`
	CreatedDate string      `json:"createdDate"`
}

// LastUpdated  contains information about the last update
type LastUpdated struct {
	By           User   `json:"by"`
	When         string `json:"when"`
	FriendlyWhen string `json:"friendlyWhen"`
	Message      string `json:"message"`
	Number       int    `json:"number"`
	MinorEdit    bool   `json:"minorEdit"`
	SyncRev      string `json:"syncRev"`
	ConfRev      string `json:"confRev"`
}

// GetHistory returns history information
func (a *API) GetHistory(ctx context.Context, id string) (*History, error) {
	ep, err := a.getContentGenericEndpoint(id, "history")
	if err != nil {
		return nil, err
	}
	return a.SendHistoryRequest(ctx, ep, "GET")
}

// Labels is the label containter type
type Labels struct {
	Labels []Label `json:"results"`
	Start  int     `json:"start,omitempty"`
	Limit  int     `json:"limit,omitempty"`
	Size   int     `json:"size,omitempty"`
}

// Label contains label information
type Label struct {
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
	Label  string `json:"label,omitempty"`
}

// GetLabels returns a list of labels attachted to a content object
func (a *API) GetLabels(ctx context.Context, id string) (*Labels, error) {
	ep, err := a.getContentGenericEndpoint(id, "label")
	if err != nil {
		return nil, err
	}
	return a.SendLabelRequest(ctx, ep, "GET", nil)
}

// AddLabels returns adds labels
func (a *API) AddLabels(ctx context.Context, id string, labels *[]Label) (*Labels, error) {
	ep, err := a.getContentGenericEndpoint(id, "label")
	if err != nil {
		return nil, err
	}
	return a.SendLabelRequest(ctx, ep, "POST", labels)
}

// DeleteLabel removes a label by name from content identified by id
func (a *API) DeleteLabel(ctx context.Context, id string, name string) (*Labels, error) {
	ep, err := a.getContentGenericEndpoint(id, "label/"+name)
	if err != nil {
		return nil, err
	}
	return a.SendLabelRequest(ctx, ep, "DELETE", nil)
}

// ContentVersionResult contains the version results
type ContentVersionResult struct {
	Result []Version `json:"results"`
}

// GetContentVersion gets all versions of this content
func (a *API) GetContentVersion(ctx context.Context, id string) (*ContentVersionResult, error) {
	ep, err := a.getContentGenericEndpoint(id, "version")
	if err != nil {
		return nil, err
	}
	return a.SendContentVersionRequest(ctx, ep, "GET")
}

// addContentQueryParams adds the defined query parameters
func addContentQueryParams(query ContentQuery) *url.Values {

	data := url.Values{}
	if len(query.Expand) != 0 {
		data.Set("expand", strings.Join(query.Expand, ","))
	}
	//get specific version
	if query.Version != 0 {
		data.Set("version", strconv.Itoa(query.Version))
	}
	if query.Limit != 0 {
		data.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.OrderBy != "" {
		data.Set("orderby", query.OrderBy)
	}
	if query.PostingDay != "" {
		data.Set("postingDay", query.PostingDay)
	}
	if query.SpaceKey != "" {
		data.Set("spaceKey", query.SpaceKey)
	}
	if query.Start != 0 {
		data.Set("start", strconv.Itoa(query.Start))
	}
	if query.Status != "" {
		data.Set("status", query.Status)
	}
	if query.Title != "" {
		data.Set("title", query.Title)
	}
	if query.Trigger != "" {
		data.Set("trigger", query.Trigger)
	}
	if query.Type != "" {
		data.Set("type", query.Type)
	}
	return &data
}
