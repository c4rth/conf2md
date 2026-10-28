package gliffy

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"
)

type objectConversion struct {
	style    string
	value    string
	geometry *MxGeometry
	vertex   bool
	edge     bool
}

func (c *converter) convertGraphiclessSwimlane(obj *Object, geometry *MxGeometry) objectConversion {
	return objectConversion{style: c.swimlaneStyle(obj), geometry: geometry, vertex: true}
}

func (c *converter) convertGraphiclessMindmap(obj *Object, geometry *MxGeometry) objectConversion {
	mindmap := obj.Children[0].Graphic.Mindmap
	var style strings.Builder
	writeStrings(&style,
		"shape=", c.translate(obj.UID, ""), ";",
		"shadow=", boolString(mindmap.DropShadow), ";strokeWidth=", format(math.Floor(mindmap.StrokeWidth+0.5)), ";",
		"fillColor=", mindmap.FillColor, ";strokeColor=", mindmap.StrokeColor, ";",
		dashStyle(mindmap.DashStyle, 1),
	)
	if mindmap.Gradient {
		writeStrings(&style, "gradientColor=#FFFFFF;gradientDirection=north;")
	}
	return objectConversion{style: style.String(), geometry: geometry, vertex: true}
}

func (c *converter) convertShapeObject(obj *Object, shape *Shape, translated string, geometry *MxGeometry) objectConversion {
	styleParts := make([]string, 0, 4)
	if translated != "" {
		styleParts = append(styleParts, "shape="+translated+";")
	}
	styleParts = append(styleParts, shapeStyle(obj, shape, strings.Join(styleParts, "")), dashStyle(shape.DashStyle, 1))

	if obj.UID == "com.gliffy.shape.flowchart.flowchart_v1.default.subroutine" && obj.Width != 0 {
		styleParts = append(styleParts, "size="+format(10/obj.Width)+";")
	}
	value := ""
	if fragment := fragmentText(obj.UID); fragment != "" {
		value = fragment
		if len(obj.Children) > 0 {
			obj.Children = obj.Children[1:]
		}
	}
	return objectConversion{style: strings.Join(styleParts, ""), value: value, geometry: geometry, vertex: true}
}

func (c *converter) convertLineObject(obj *Object, geometry *MxGeometry) objectConversion {
	line := obj.Graphic.Line
	for _, arrow := range []*int{line.StartArrow, line.EndArrow} {
		if arrow != nil && (*arrow < 0 || *arrow > 19) {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s: unsupported arrow %d; using no arrow", obj.ID, *arrow))
		}
	}
	if line.InterpolationType != "" && line.InterpolationType != "linear" &&
		line.InterpolationType != "orthogonal" && line.InterpolationType != "quadratic" {
		c.warnings = append(c.warnings, fmt.Sprintf("object %s: unsupported line interpolation %q; using linear", obj.ID, line.InterpolationType))
	}
	style := strings.Join([]string{
		"shape=filledEdge;",
		"strokeWidth=" + format(math.Floor(line.StrokeWidth+0.5)) + ";",
		"strokeColor=" + nonempty(line.StrokeColor, "#000000") + ";",
		"fillColor=" + nonempty(line.FillColor, "none") + ";",
		arrowStyle(line.StartArrow, true),
		arrowStyle(line.EndArrow, false),
		"rounded=" + boolString(line.CornerRadius != nil) + ";",
		dashStyle(line.DashStyle, math.Floor(line.StrokeWidth+0.5)),
		lineStyle(line.InterpolationType),
	}, "")
	geometry.X, geometry.Y, geometry.Relative = "0", "0", "1"
	return objectConversion{style: style, geometry: geometry, edge: true}
}

func (c *converter) convertTextObject(obj *Object, graphic *Graphic, geometry *MxGeometry) objectConversion {
	var style strings.Builder
	writeStrings(&style, "text;html=1;nl2Br=0;")
	if obj.parent != nil && !obj.parent.isGroup() {
		if obj.parent.isLine() {
			geometry = lineLabelGeometry(obj)
			graphic.Text.halign = ""
			writeStrings(&style, "labelBackgroundColor=", c.diagram.Stage.Background, ";")
		} else {
			geometry.X, geometry.Y = "0", "0"
			geometry.Width, geometry.Height = obj.parent.cell.Geometry.Width, obj.parent.cell.Geometry.Height
		}
		geometry.Relative = "1"
	}
	return objectConversion{style: style.String(), geometry: geometry, vertex: true}
}

func convertImageObject(graphic *Graphic, geometry *MxGeometry, translated string) objectConversion {
	style := "shape=" + nonempty(translated, "image") + ";image=" + strings.ReplaceAll(graphic.Image.URL, ";base64", "") + ";"
	return objectConversion{style: style, geometry: geometry, vertex: true}
}

func (c *converter) convertSVGObject(obj *Object, graphic *Graphic, geometry *MxGeometry) objectConversion {
	var style strings.Builder
	writeStrings(&style, "shape=image;imageAspect=0;")
	if data := c.resourceData(graphic.SVG.EmbeddedResourceID); data != "" {
		if strings.Contains(data, "viewBox=") {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s: SVG viewBox preserved; automatic artwork cropping requires an SVG renderer", obj.ID))
		}
		writeStrings(&style, "image=data:image/svg+xml,", base64.StdEncoding.EncodeToString([]byte(data)), ";")
	}
	return objectConversion{style: style.String(), geometry: geometry, vertex: true}
}

func (c *converter) finishObjectConversion(obj *Object, converted objectConversion) objectConversion {
	if !converted.edge {
		converted.style = rotateObject(obj, converted.geometry, converted.style)
	}
	if obj.isGroup() {
		converted.style += "gliffyId=" + obj.ID + ";"
		return converted
	}
	if text := findText(obj); text != nil {
		txt := text.Graphic.Text
		if obj.isSwimlane() {
			txt.forceTopPaddingShift, txt.Valign = true, "middle"
		}
		converted.value = objectText(text)
		x, y := text.X, text.Y
		if shifts := obj.coordinateShifts(); len(shifts) == 4 {
			dx, dy, _, _ := obj.adjustShifts(shifts)
			x, y = x-dx, y-dy
		}
		if text == obj || strings.Contains(obj.UID, "chevron") {
			x, y = 0, 0
		}
		textStyleValue := textStyle(txt, x, y)
		if strings.Contains(obj.UID, "com.gliffy.shape.uml.uml_v2.activity.frame") {
			converted.style = strings.ReplaceAll(converted.style, "labelX=32", "labelX="+format(float64(float32(text.Width)*float32(1.1))))
			textStyleValue = strings.ReplaceAll(textStyle(txt, 0, 0), "verticalAlign=middle", "verticalAlign=top")
		}
		converted.style += "html=1;nl2Br=0;"
		if !converted.edge {
			converted.style += textStyleValue
		} else {
			converted.value = ""
		}
	}
	converted.style += "gliffyId=" + obj.ID + ";"
	return converted
}
