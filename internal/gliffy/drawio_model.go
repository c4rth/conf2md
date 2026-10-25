package gliffy

import "encoding/xml"

type MxGraphModel struct {
	XMLName    xml.Name `xml:"mxGraphModel"`
	Background string   `xml:"background,attr"`
	Grid       string   `xml:"grid,attr"`
	Guides     string   `xml:"guides,attr"`
	Style      string   `xml:"style,attr"`
	Root       MxRoot   `xml:"root"`
}

type MxRoot struct {
	Cells []MxCell `xml:"mxCell"`
}

type MxCell struct {
	ID         string        `xml:"id,attr"`
	Parent     string        `xml:"parent,attr,omitempty"`
	Style      string        `xml:"style,attr,omitempty"`
	Value      string        `xml:"value,attr,omitempty"`
	HasValue   bool          `xml:"-"`
	Vertex     string        `xml:"vertex,attr,omitempty"`
	HasVertex  bool          `xml:"-"`
	Edge       string        `xml:"edge,attr,omitempty"`
	HasEdge    bool          `xml:"-"`
	Visible    string        `xml:"visible,attr,omitempty"`
	HasVisible bool          `xml:"-"`
	Extra      []xml.Attr    `xml:",attr"`
	Geometry   *MxGeometry   `xml:"mxGeometry,omitempty"`
	UserObject *MxUserObject `xml:"-"`
}

type MxUserObject struct {
	Link    string `xml:"link,attr,omitempty"`
	Label   string `xml:"label,attr,omitempty"`
	Tooltip string `xml:"tooltip,attr,omitempty"`
}

type MxGeometry struct {
	XMLName     xml.Name      `xml:"mxGeometry"`
	As          string        `xml:"as,attr"`
	X           string        `xml:"x,attr,omitempty"`
	Y           string        `xml:"y,attr,omitempty"`
	Width       string        `xml:"width,attr,omitempty"`
	Height      string        `xml:"height,attr,omitempty"`
	Relative    string        `xml:"relative,attr,omitempty"`
	Points      *MxPointArray `xml:"-"`
	SourcePoint *MxPoint      `xml:"-"`
	TargetPoint *MxPoint      `xml:"-"`
	Offset      *MxPoint      `xml:"-"`
}

type MxPoint struct {
	X  string `xml:"x,attr"`
	Y  string `xml:"y,attr"`
	As string `xml:"as,attr,omitempty"`
}

type MxPointArray struct {
	As     string    `xml:"as,attr"`
	Points []MxPoint `xml:"mxPoint"`
}
