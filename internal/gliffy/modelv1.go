package gliffy

import (
	"encoding/json"
	"fmt"
)

// GliffyDiagramV1 Gliffy version 1.x JSON diagrams
type GliffyDiagramV1 struct {
	ContentType       string            `json:"contentType,omitempty"`
	Version           string            `json:"version,omitempty"`
	Metadata          Metadata          `json:"metadata,omitempty"`
	EmbeddedResources EmbeddedResources `json:"embeddedResources,omitempty"`
	Stage             Stage             `json:"stage,omitempty"`
}

//

type Metadata struct {
	Title string `json:"title,omitempty"`
}

type EmbeddedResources struct {
	Resources []Resource `json:"resources,omitempty"`
}

type Resource struct {
	ID       string `json:"id,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Stage struct {
	Background      string     `json:"background,omitempty"`
	Width           float64    `json:"width,omitempty"`
	Height          float64    `json:"height,omitempty"`
	AutoFit         bool       `json:"autoFit,omitempty"`
	GridOn          bool       `json:"gridOn,omitempty"`
	DrawingGuidesOn bool       `json:"drawingGuidesOn,omitempty"`
	Objects         []*Object  `json:"objects,omitempty"`
	Layers          []Layer    `json:"layers,omitempty"`
	TextStyles      TextStyles `json:"textStyles,omitempty"`
}

type TextStyles struct {
	Global *GlobalTextStyles `json:"global,omitempty"`
}

type GlobalTextStyles struct {
	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
}

type Layer struct {
	GUID    string `json:"guid,omitempty"`
	Order   int    `json:"order,omitempty"`
	Name    string `json:"name,omitempty"`
	Active  bool   `json:"active,omitempty"`
	Locked  bool   `json:"locked,omitempty"`
	Visible bool   `json:"visible,omitempty"`
}

type Object struct {
	X              float64         `json:"x,omitempty"`
	Y              float64         `json:"y,omitempty"`
	ID             string          `json:"id,omitempty"`
	Width          float64         `json:"width,omitempty"`
	Height         float64         `json:"height,omitempty"`
	Rotation       float64         `json:"rotation,omitempty"`
	UID            string          `json:"uid,omitempty"`
	TID            string          `json:"tid,omitempty"`
	Order          json.RawMessage `json:"order,omitempty"`
	LayerID        string          `json:"layerId,omitempty"`
	Graphic        *Graphic        `json:"graphic,omitempty"`
	Children       []*Object       `json:"children,omitempty"`
	Constraints    *Constraints    `json:"constraints,omitempty"`
	LockShape      bool            `json:"lockshape,omitempty"`
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
	Type      GraphicType `json:"type,omitempty"`
	Text      *Text       `json:"Text,omitempty"`
	Line      *Line       `json:"Line,omitempty"`
	Link      *Link       `json:"Link,omitempty"`
	Shape     *Shape      `json:"Shape,omitempty"`
	Image     *Image      `json:"Image,omitempty"`
	SVG       *SVG        `json:"Svg,omitempty"`
	Mindmap   *Shape      `json:"Mindmap,omitempty"`
	PopupNote *PopupNote  `json:"PopupNote,omitempty"`
}

type Shape struct {
	StrokeColor string  `json:"strokeColor,omitempty"`
	FillColor   string  `json:"fillColor,omitempty"`
	DashStyle   string  `json:"dashStyle,omitempty"`
	StrokeWidth float64 `json:"strokeWidth,omitempty"`
	TID         string  `json:"tid,omitempty"`
	Gradient    bool    `json:"gradient,omitempty"`
	DropShadow  bool    `json:"dropShadow,omitempty"`
	Opacity     float64 `json:"opacity,omitempty"`
}

type Line struct {
	StrokeColor       string      `json:"strokeColor,omitempty"`
	FillColor         string      `json:"fillColor,omitempty"`
	DashStyle         string      `json:"dashStyle,omitempty"`
	StrokeWidth       float64     `json:"strokeWidth,omitempty"`
	StartArrow        *int        `json:"startArrow,omitempty"`
	EndArrow          *int        `json:"endArrow,omitempty"`
	InterpolationType string      `json:"interpolationType,omitempty"`
	CornerRadius      *float64    `json:"cornerRadius,omitempty"`
	ControlPath       [][]float64 `json:"controlPath,omitempty"`
}

type Image struct {
	URL string `json:"url,omitempty"`
}

type SVG struct {
	EmbeddedResourceID *string `json:"embeddedResourceId,omitempty"`
}

type PopupNote struct {
	Text string `json:"text,omitempty"`
}

type Link struct {
	Href       string `json:"href,omitempty"`
	RenderIcon bool   `json:"renderIcon,omitempty"`
}

type Text struct {
	LineTValue           *float64 `json:"lineTValue,omitempty"`
	LinePerpValue        *float64 `json:"linePerpValue,omitempty"`
	CardinalityType      string   `json:"cardinalityType,omitempty"`
	Overflow             string   `json:"overflow,omitempty"`
	HTML                 string   `json:"html,omitempty"`
	Valign               string   `json:"valign,omitempty"`
	VPosition            string   `json:"vposition,omitempty"`
	HPosition            string   `json:"hposition,omitempty"`
	PaddingLeft          *float64 `json:"paddingLeft,omitempty"`
	PaddingRight         *float64 `json:"paddingRight,omitempty"`
	PaddingBottom        *float64 `json:"paddingBottom,omitempty"`
	PaddingTop           *float64 `json:"paddingTop,omitempty"`
	halign               string
	forceTopPaddingShift bool
}

type Constraints struct {
	StartConstraint *Constraint `json:"startConstraint,omitempty"`
	EndConstraint   *Constraint `json:"endConstraint,omitempty"`
}

type Constraint struct {
	Start *ConstraintData `json:"StartPositionConstraint,omitempty"`
	End   *ConstraintData `json:"EndPositionConstraint,omitempty"`
}

type ConstraintData struct {
	NodeID string  `json:"nodeId,omitempty"`
	PX     float64 `json:"px,omitempty"`
	PY     float64 `json:"py,omitempty"`
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

func (r *Resource) UnmarshalJSON(data []byte) error {
	type plain Resource
	var decoded struct {
		*plain
		ID json.RawMessage `json:"id"`
	}
	decoded.plain = (*plain)(r)
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	id, err := identifier(decoded.ID)
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

func (s *SVG) UnmarshalJSON(data []byte) error {
	type plain SVG
	var decoded struct {
		*plain
		EmbeddedResourceID json.RawMessage `json:"embeddedResourceId"`
	}
	decoded.plain = (*plain)(s)
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	s.EmbeddedResourceID = nil
	if len(decoded.EmbeddedResourceID) == 0 || string(decoded.EmbeddedResourceID) == "null" {
		return nil
	}
	id, err := identifier(decoded.EmbeddedResourceID)
	if err != nil {
		return err
	}
	s.EmbeddedResourceID = &id
	return nil
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
