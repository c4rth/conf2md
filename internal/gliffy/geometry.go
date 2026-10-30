package gliffy

import (
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var coordinateFixes = map[string][]float64{
	"com.gliffy.shape.flowchart.flowchart_v1.default.paper_tape":        {0, -.1, 0, .2},
	"com.gliffy.shape.uml.uml_v1.default.node":                          {0, -10, 10, 10},
	"com.gliffy.shape.uml.uml_v2.deployment.node":                       {0, -10, 10, 10},
	"com.gliffy.shape.uml.uml_v2.deployment.device_node":                {0, -10, 10, 10},
	"com.gliffy.shape.uml.uml_v2.deployment.execution_environment_node": {0, -10, 10, 10},
	"com.gliffy.shape.flowchart.flowchart_v1.default.data_storage":      {0, 0, .115, 0},
	"com.gliffy.shape.flowchart.flowchart_v1.default.database":          {0, 0, 0, .15},
	"com.gliffy.stencil.entity_lifeline.uml_v2":                         {10, 0, -20, 0},
	"com.gliffy.stencil.boundary_lifeline.uml_v2":                       {35, 0, -70, 0},
	"com.gliffy.stencil.control_lifeline.uml_v2":                        {10, 0, -20, 0},
	"com.gliffy.shape.ui.ui_v3.containers_content.browser":              {0, -40, 0, 40, 1},
}

func (o *Object) coordinateShifts() []float64 {
	key := o.UID
	if key == "" && o.Graphic != nil && o.Graphic.Shape != nil {
		key = o.Graphic.Shape.TID
	}
	return coordinateFixes[key]
}

func (o *Object) adjustShifts(fix []float64) (dx, dy, dw, dh float64) {
	shift := func(value, size float64) float64 {
		if math.Abs(value) < 1 {
			return value * size
		}
		return value
	}
	dx, dy = shift(fix[0], o.Width), shift(fix[1], o.Height)
	dw, dh = shift(fix[2], o.Width), shift(fix[3], o.Height)
	if o.Rotation > 0 {
		x1, y1 := rotatedGeometry(o.X, o.Y, o.Width, o.Height, o.Rotation, 0, 0)
		x2, y2 := rotatedGeometry(o.X+dx, o.Y+dy, o.Width+dw, o.Height+dh, o.Rotation, 0, 0)
		dx, dy = dx+x2-x1, dy+y2-y1
	}
	return
}

func (o *Object) adjustGeometry(geo *MxGeometry) {
	if fix := o.coordinateShifts(); fix != nil {
		dx, dy, dw, dh := o.adjustShifts(fix)
		geo.X, geo.Y, geo.Width, geo.Height = format(o.X+dx), format(o.Y+dy), format(o.Width+dw), format(o.Height+dh)
	}
}

func rotatePoint(x, y, angle, cx, cy float64) (float64, float64) {
	sin, cos := math.Sincos(angle * math.Pi / 180)
	return (x-cx)*cos - (y-cy)*sin + cx, (x-cx)*sin + (y-cy)*cos + cy
}

func rotatedGeometry(x, y, w, h, angle, cx, cy float64) (float64, float64) {
	x, y = rotatePoint(x+w/2, y+h/2, angle, cx, cy)
	return math.Floor(x - w/2 + .5), math.Floor(y - h/2 + .5)
}

var rotationPattern = regexp.MustCompile(`rotation=(-?\d+(?:\.\d+)?)`)

func rotateObject(obj *Object, geometry *MxGeometry, style string) string {
	if match := rotationPattern.FindStringSubmatch(style); match != nil {
		initial, _ := strconv.ParseFloat(match[1], 64)
		return strings.Replace(style, match[0], "rotation="+format(initial+obj.Rotation), 1)
	}
	if obj.Rotation == 0 {
		return style
	}
	if strings.Contains(style, "swimlane;collapsible=0;") && obj.Rotation == 270 {
		w, _ := strconv.ParseFloat(geometry.Width, 64)
		h, _ := strconv.ParseFloat(geometry.Height, 64)
		x, _ := strconv.ParseFloat(geometry.X, 64)
		y, _ := strconv.ParseFloat(geometry.Y, 64)
		geometry.X, geometry.Y = format(x+(w-h)/2), format(y+(h-w)/2)
		geometry.Width, geometry.Height = format(h), format(w)
		return style + "childLayout=stackLayout;resizeParent=1;resizeParentMax=0;horizontal=0;horizontalStack=0;"
	}
	if obj.isGroup() {
		for _, child := range obj.Children {
			if child != nil {
				child.X, child.Y = rotatePoint(child.X, child.Y, obj.Rotation,
					obj.Width/2-child.Width/2, obj.Height/2-child.Height/2)
				child.Rotation += obj.Rotation
			}
		}
	}
	return style + "rotation=" + format(obj.Rotation) + ";"
}

func lineLabelGeometry(obj *Object) *MxGeometry {
	text, line := obj.Graphic.Text, obj.parent.Graphic.Line
	t := .5
	if text.LineTValue != nil {
		t = *text.LineTValue
	}
	x, y := 0.0, 0.0
	points := line.ControlPath
	if text.LinePerpValue != nil && len(points) >= 2 {
		i, j := 0, len(points)-1
		switch text.CardinalityType {
		case "begin":
			j = 1
		case "end":
			i = len(points) - 2
		}
		if len(points[i]) >= 2 && len(points[j]) >= 2 {
			if (text.CardinalityType != "begin" && text.CardinalityType != "end") || points[i][1] == points[j][1] {
				y = *text.LinePerpValue
				if points[i][0] > points[j][0] {
					y = -y
				}
			} else {
				x = *text.LinePerpValue
				if points[i][1] < points[j][1] {
					x = -x
				}
			}
		}
	}
	return &MxGeometry{As: "geometry", X: format(t*2 - 1), Y: "0", Width: "0", Height: "0", Relative: "1",
		Offset: &MxPoint{X: format(x), Y: format(y), As: "offset"}}
}

func constraintData(obj *Object, start bool) *ConstraintData {
	if obj.Constraints == nil {
		return nil
	}
	if start && obj.Constraints.StartConstraint != nil {
		return obj.Constraints.StartConstraint.Start
	}
	if !start && obj.Constraints.EndConstraint != nil {
		return obj.Constraints.EndConstraint.End
	}
	return nil
}

func styleValue(style, key string) string {
	for part := range strings.SplitSeq(style, ";") {
		if k, value, ok := strings.Cut(part, "="); ok && strings.EqualFold(k, key) {
			return value
		}
	}
	return "east"
}

func (c *converter) setWaypoints(obj *Object) {
	cell, geo := obj.cell, obj.cell.Geometry
	var terminals [2]*Object
	for i, start := range []bool{true, false} {
		if data := constraintData(obj, start); data != nil {
			terminal := c.vertices[data.NodeID]
			if terminal != nil && terminal.cell != nil && terminal.cellID != "" {
				terminals[i] = terminal
				key := "target"
				if start {
					key = "source"
				}
				cell.Extra = append(cell.Extra, xml.Attr{Name: xml.Name{Local: key}, Value: terminal.cellID})
			}
		}
	}
	geo.Relative = "1"
	path := obj.Graphic.Line.ControlPath
	if len(path) < 2 {
		return
	}
	type point struct{ x, y float64 }
	points := make([]point, 0, len(path))
	for _, p := range path {
		if len(p) < 2 {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s: control point needs x and y", obj.ID))
			return
		}
		x, y := p[0]+obj.X, p[1]+obj.Y
		if obj.Rotation != 0 {
			x, y = rotatePoint(x, y, obj.Rotation, obj.X+obj.Width/2, obj.Y+obj.Height/2)
		}
		points = append(points, point{x, y})
	}
	orthogonal := true
	for i := 1; i < len(points); i++ {
		orthogonal = orthogonal && (points[i-1].x == points[i].x || points[i-1].y == points[i].y)
	}
	first, last := points[0], points[len(points)-1]
	for i, start := range []bool{true, false} {
		terminal := terminals[i]
		if terminal == nil {
			if start {
				geo.SourcePoint = &MxPoint{X: format(first.x), Y: format(first.y), As: "sourcePoint"}
				points = points[1:]
			} else {
				geo.TargetPoint = &MxPoint{X: format(last.x), Y: format(last.y), As: "targetPoint"}
				points = points[:len(points)-1]
			}
			continue
		}
		if data := constraintData(obj, start); data != nil {
			x, y := data.PX, data.PY
			angle := map[string]float64{"south": 270, "west": 180, "north": 90}[strings.ToLower(styleValue(terminal.cell.Style, "direction"))]
			if angle != 0 {
				x, y = rotatePoint(x, y, angle, .5, .5)
			}
			if !orthogonal || (x == .5 && y == .5) || obj.UID == "com.gliffy.shape.uml.uml_v2.class.association" {
				key := "entry"
				if start {
					key = "exit"
				}
				cell.Style += key + "X=" + format(x) + ";" + key + "Y=" + format(y) + ";" + key + "Perimeter=0;"
			}
		}
	}
	if orthogonal {
		cell.Style += "edgeStyle=orthogonalEdgeStyle;"
		deduplicated := make([]point, 0, len(points))
		for _, p := range points {
			if len(deduplicated) == 0 || deduplicated[len(deduplicated)-1] != p {
				deduplicated = append(deduplicated, p)
			}
		}
		points = deduplicated
		if len(points) == 0 && ((terminals[0] == nil) != (terminals[1] == nil)) {
			center := point{first.x + (last.x-first.x)/2, first.y + (last.y-first.y)/2}
			points = []point{center, center}
		}
	}
	if len(points) > 0 {
		geo.Points = &MxPointArray{As: "points"}
		for _, p := range points {
			geo.Points.Points = append(geo.Points.Points, MxPoint{X: format(p.x), Y: format(p.y)})
		}
	}
}
