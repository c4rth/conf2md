package gliffy

import (
	"encoding/json"
	"fmt"
)

// Diagram is the top-level representation of a Gliffy JSON diagram.
type Diagram struct {
	ContentType       string            `json:"contentType"`
	Version           string            `json:"version"`
	Metadata          Metadata          `json:"metadata"`
	EmbeddedResources EmbeddedResources `json:"embeddedResources"`
	Stage             Stage             `json:"stage"`
}

type Metadata struct {
	Title string `json:"title"`
}

type EmbeddedResources struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	ID       int    `json:"id"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type Stage struct {
	Background      string     `json:"background"`
	Width           float64    `json:"width"`
	Height          float64    `json:"height"`
	AutoFit         bool       `json:"autoFit"`
	GridOn          bool       `json:"gridOn"`
	DrawingGuidesOn bool       `json:"drawingGuidesOn"`
	Objects         []*Object  `json:"objects"`
	Layers          []Layer    `json:"layers"`
	TextStyles      TextStyles `json:"textStyles"`
}

type TextStyles struct {
	Global *GlobalTextStyles `json:"global"`
}

type GlobalTextStyles struct {
	Size  string `json:"size"`
	Color string `json:"color"`
}

type Layer struct {
	GUID    string `json:"guid"`
	Order   int    `json:"order"`
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Locked  bool   `json:"locked"`
	Visible bool   `json:"visible"`
}

type Object struct {
	X              float64         `json:"x"`
	Y              float64         `json:"y"`
	ID             string          `json:"id"`
	Width          float64         `json:"width"`
	Height         float64         `json:"height"`
	Rotation       float64         `json:"rotation"`
	UID            string          `json:"uid"`
	TID            string          `json:"tid"`
	Order          json.RawMessage `json:"order"`
	LayerID        string          `json:"layerId"`
	Graphic        *Graphic        `json:"graphic"`
	Children       []*Object       `json:"children"`
	Constraints    *Constraints    `json:"constraints"`
	LockShape      bool            `json:"lockshape"`
	parent         *Object
	cellID         string
	cell           *MxCell
	importChildren []*Object
}

type GraphicType string

const (
	GraphicTypeShape     GraphicType = "Shape"
	GraphicTypeLine      GraphicType = "Line"
	GraphicTypeText      GraphicType = "Text"
	GraphicTypeImage     GraphicType = "Image"
	GraphicTypeSVG       GraphicType = "Svg"
	GraphicTypeLink      GraphicType = "Link"
	GraphicTypePopupNote GraphicType = "PopupNote"
	GraphicTypeMindmap   GraphicType = "Mindmap"
)

type Graphic struct {
	Type      GraphicType `json:"type"`
	Text      *Text       `json:"Text"`
	Line      *Line       `json:"Line"`
	Link      *Link       `json:"Link"`
	Shape     *Shape      `json:"Shape"`
	Image     *Image      `json:"Image"`
	SVG       *SVG        `json:"Svg"`
	Mindmap   *Shape      `json:"Mindmap"`
	PopupNote *PopupNote  `json:"PopupNote"`
}

type Shape struct {
	StrokeColor string  `json:"strokeColor"`
	FillColor   string  `json:"fillColor"`
	DashStyle   string  `json:"dashStyle"`
	StrokeWidth float64 `json:"strokeWidth"`
	TID         string  `json:"tid"`
	Gradient    bool    `json:"gradient"`
	DropShadow  bool    `json:"dropShadow"`
	Opacity     float64 `json:"opacity"`
}

type Line struct {
	StrokeColor       string      `json:"strokeColor"`
	FillColor         string      `json:"fillColor"`
	DashStyle         string      `json:"dashStyle"`
	StrokeWidth       float64     `json:"strokeWidth"`
	StartArrow        *int        `json:"startArrow"`
	EndArrow          *int        `json:"endArrow"`
	InterpolationType string      `json:"interpolationType"`
	CornerRadius      *float64    `json:"cornerRadius"`
	ControlPath       [][]float64 `json:"controlPath"`
}

type Image struct {
	URL string `json:"url"`
}

type SVG struct {
	EmbeddedResourceID *int `json:"embeddedResourceId"`
}

type PopupNote struct {
	Text string `json:"text"`
}

type Link struct {
	Href       string `json:"href"`
	RenderIcon bool   `json:"renderIcon"`
}

type Text struct {
	LineTValue           *float64 `json:"lineTValue"`
	LinePerpValue        *float64 `json:"linePerpValue"`
	CardinalityType      string   `json:"cardinalityType"`
	Overflow             string   `json:"overflow"`
	HTML                 string   `json:"html"`
	Valign               string   `json:"valign"`
	VPosition            string   `json:"vposition"`
	HPosition            string   `json:"hposition"`
	PaddingLeft          *float64 `json:"paddingLeft"`
	PaddingRight         *float64 `json:"paddingRight"`
	PaddingBottom        *float64 `json:"paddingBottom"`
	PaddingTop           *float64 `json:"paddingTop"`
	halign               string
	forceTopPaddingShift bool
}

type Constraints struct {
	StartConstraint *Constraint `json:"startConstraint"`
	EndConstraint   *Constraint `json:"endConstraint"`
}

type Constraint struct {
	Start *ConstraintData `json:"StartPositionConstraint"`
	End   *ConstraintData `json:"EndPositionConstraint"`
}

type ConstraintData struct {
	NodeID string  `json:"nodeId"`
	PX     float64 `json:"px"`
	PY     float64 `json:"py"`
}

// Gliffy exports use both UUID strings and numeric node identifiers.
func identifier(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var value string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", err
		}
		return value, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", fmt.Errorf("invalid Gliffy identifier %s: %w", raw, err)
	}
	return number.String(), nil
}

func (o *Object) UnmarshalJSON(data []byte) error {
	type plain Object
	var decoded struct {
		*plain
		ID json.RawMessage `json:"id"`
	}
	decoded.plain = (*plain)(o)
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	id, err := identifier(decoded.ID)
	if err != nil {
		return err
	}
	o.ID = id
	return nil
}

func (c *ConstraintData) UnmarshalJSON(data []byte) error {
	type plain ConstraintData
	var decoded struct {
		*plain
		NodeID json.RawMessage `json:"nodeId"`
	}
	decoded.plain = (*plain)(c)
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	id, err := identifier(decoded.NodeID)
	if err != nil {
		return err
	}
	c.NodeID = id
	return nil
}
