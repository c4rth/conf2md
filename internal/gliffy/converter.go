package gliffy

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed gliffyTranslation.properties
var translationProperties string

// Result contains the Draw.io graph model and conversion warnings.
type Result struct {
	XML      string
	Warnings []string
}

// Convert decodes a Gliffy diagram and converts it to a Draw.io graph model.
func Convert(input []byte) (Result, error) {
	var diagram Diagram
	if err := json.Unmarshal(input, &diagram); err != nil {
		return Result{}, fmt.Errorf("decode Gliffy diagram: %w", err)
	}
	if diagram.Stage.Objects == nil {
		return Result{}, errors.New("decode Gliffy diagram: stage.objects is missing")
	}
	if diagram.Stage.Background == "" {
		diagram.Stage.Background = "#FFFFFF"
	}

	converter := converter{
		diagram:      &diagram,
		translations: loadTranslations(),
		layerIDs:     make(map[string]string),
		vertices:     make(map[string]*Object),
	}
	converter.prepare(diagram.Stage.Objects, nil)
	converter.createLayers()
	converter.collect(diagram.Stage.Objects, nil)
	sortObjects(diagram.Stage.Objects)
	converter.assignObjectIDs(diagram.Stage.Objects)
	rootParent := "2"
	if len(diagram.Stage.Layers) == 0 {
		rootParent = "1"
	}
	for _, obj := range diagram.Stage.Objects {
		converter.addObject(obj, rootParent)
	}

	model := MxGraphModel{
		Background: diagram.Stage.Background,
		Grid:       boolString(diagram.Stage.GridOn),
		Guides:     boolString(diagram.Stage.DrawingGuidesOn),
		Style:      "default-style2",
		Root:       MxRoot{Cells: converter.cells},
	}
	data, err := xml.Marshal(model)
	if err != nil {
		return Result{}, fmt.Errorf("encode Draw.io graph model: %w", err)
	}
	return Result{XML: string(data), Warnings: converter.warnings}, nil
}

type converter struct {
	diagram      *Diagram
	translations map[string]string
	cells        []MxCell
	layerIDs     map[string]string
	nextID       int
	warnings     []string
	vertices     map[string]*Object
}

func loadTranslations() map[string]string {
	translations := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(translationProperties))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		separator := strings.IndexAny(line, "=:")
		if separator < 0 {
			continue
		}
		translations[strings.TrimSpace(line[:separator])] = strings.TrimSpace(line[separator+1:])
	}
	return translations
}

func (c *converter) prepare(objects []*Object, parent *Object) {
	for _, obj := range objects {
		if obj == nil {
			c.warnings = append(c.warnings, "null Gliffy object ignored")
			continue
		}
		obj.parent = parent
		c.prepare(obj.Children, obj)
		if obj.Graphic != nil && obj.Graphic.Text != nil {
			txt := obj.Graphic.Text
			matches := alignmentPattern.FindAllStringSubmatch(txt.HTML, -1)
			if len(matches) > 0 {
				txt.halign = matches[len(matches)-1][1]
			}
			txt.HTML = textHTML(txt.HTML)
		}
		if obj.isGroup() {
			obj.normalizeGroup()
		}
	}
}

func sortObjects(objects []*Object) {
	sort.SliceStable(objects, func(i, j int) bool {
		if objects[i] == nil || objects[j] == nil {
			return objects[i] != nil
		}
		lraw, rraw := rawOrder(objects[i].Order), rawOrder(objects[j].Order)
		if lraw == "" || lraw == "null" {
			return false
		}
		if rraw == "" || rraw == "null" {
			return true
		}
		left, leftOK := objectOrder(objects[i].Order)
		right, rightOK := objectOrder(objects[j].Order)
		if leftOK && rightOK {
			return left < right
		}
		return lraw < rraw
	})
}

func objectOrder(raw json.RawMessage) (float64, bool) {
	value := strings.Trim(string(raw), `" `)
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func rawOrder(raw json.RawMessage) string {
	return strings.Trim(string(raw), `" `)
}

func (c *converter) createLayers() {
	layers := append([]Layer(nil), c.diagram.Stage.Layers...)
	sort.SliceStable(layers, func(i, j int) bool { return layers[i].Order < layers[j].Order })
	c.cells = append(c.cells, MxCell{ID: "0"})
	if len(layers) == 0 {
		c.cells = append(c.cells, MxCell{ID: "1", Parent: "0"})
		c.nextID = 2
		return
	}
	c.nextID = 2
	for _, layer := range layers {
		id := c.id()
		style := ""
		if layer.Locked {
			style = "locked=1;"
		}
		c.layerIDs[layer.GUID] = id
		c.cells = append(c.cells, MxCell{
			ID: id, Parent: "0", Value: layer.Name, HasValue: true,
			Style: style, Visible: boolString(layer.Visible), HasVisible: true,
		})
	}
}

func (c *converter) assignObjectIDs(objects []*Object) {
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		if obj.cell != nil {
			obj.cellID = c.id()
			obj.cell.ID = obj.cellID
		}
		c.assignObjectIDs(obj.importChildren)
	}
}

func (c *converter) id() string {
	id := strconv.Itoa(c.nextID)
	c.nextID++
	return id
}

func (c *converter) addObject(obj *Object, inheritedParent string) {
	if obj == nil {
		return
	}
	if obj.cell == nil {
		return
	}

	parentID := inheritedParent
	if obj.parent == nil && obj.LayerID != "" {
		if layerID := c.layerIDs[obj.LayerID]; layerID != "" {
			parentID = layerID
		}
	}

	cell := obj.cell
	cell.Parent = parentID
	if obj.isLine() {
		c.setWaypoints(obj)
	}
	c.cells = append(c.cells, *cell)
	for _, child := range obj.importChildren {
		c.addObject(child, obj.cellID)
	}
}

func (c *converter) collect(objects []*Object, parent *Object) {
	for _, obj := range objects {
		if obj == nil || isLink(obj) || isPopupNote(obj) {
			continue
		}
		obj.parent = parent
		style, value, geometry, vertex, edge := c.convertObject(obj)
		if !vertex && !edge {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s has an unsupported graphic type", obj.ID))
			continue
		}
		cell := &MxCell{Style: style, Value: value, HasValue: value != "",
			Geometry: geometry, Vertex: "1", HasVertex: vertex, Edge: "1", HasEdge: edge}
		obj.cell = cell
		c.decorate(obj)
		if !obj.isLine() {
			c.vertices[obj.ID] = obj
		}
		switch {
		case obj.isSwimlane() && obj.hasChildren():
			c.convertLanes(obj)
		case obj.isGroup() || obj.isSelection() || obj.isLine():
			c.collect(obj.Children, obj)
			sortObjects(obj.Children)
			obj.importChildren = obj.Children
		}
	}
}

func (c *converter) convertObject(obj *Object) (string, string, *MxGeometry, bool, bool) {
	g := obj.effectiveGraphic()
	shape := (*Shape)(nil)
	if g != nil {
		shape = g.Shape
		if shape == nil {
			shape = g.Mindmap
		}
	}
	styleParts := make([]string, 0, 20)
	value := ""
	vertex, edge := false, false
	geometry := &MxGeometry{As: "geometry", X: format(obj.X), Y: format(obj.Y), Width: format(obj.Width), Height: format(obj.Height)}
	obj.adjustGeometry(geometry)
	translated := ""
	if shape != nil {
		translated = c.translate(obj.UID, shape.TID)
	}
	if obj.isGroup() && (g == nil || translated == "") {
		styleParts = append(styleParts, "group;")
		vertex = true
	}

	switch {
	case g == nil && obj.isSwimlane() && obj.hasChildren():
		styleParts = append(styleParts, c.swimlaneStyle(obj))
		vertex = true
	case g == nil && obj.isMindmap() && obj.hasChildren() && obj.Children[0] != nil && obj.Children[0].Graphic != nil && obj.Children[0].Graphic.Mindmap != nil:
		mindmap := obj.Children[0].Graphic.Mindmap
		styleParts = append(styleParts, "shape="+c.translate(obj.UID, "")+";",
			"shadow="+boolInt(mindmap.DropShadow)+";strokeWidth="+format(math.Floor(mindmap.StrokeWidth+0.5))+";",
			"fillColor="+mindmap.FillColor+";strokeColor="+mindmap.StrokeColor+";", dashStyle(mindmap.DashStyle, 1))
		if mindmap.Gradient {
			styleParts = append(styleParts, "gradientColor=#FFFFFF;gradientDirection=north;")
		}
		vertex = true
	case g == nil:
		if !vertex {
			return "", "", nil, false, false
		}
	case (g.Type == "Shape" || g.Type == "Mindmap") && shape != nil:
		vertex = true
		if translated != "" {
			styleParts = append(styleParts, "shape="+translated+";")
		}
		styleParts = append(styleParts, shapeStyle(obj, shape, strings.Join(styleParts, "")))
		styleParts = append(styleParts, dashStyle(shape.DashStyle, 1))
		if obj.UID == "com.gliffy.shape.flowchart.flowchart_v1.default.subroutine" && obj.Width != 0 {
			styleParts = append(styleParts, "size="+format(10/obj.Width)+";")
		}
		if fragment := fragmentText(obj.UID); fragment != "" {
			value = fragment
			if len(obj.Children) > 0 {
				obj.Children = obj.Children[1:]
			}
		}
	case g.Type == "Line" && g.Line != nil:
		edge = true
		line := g.Line
		for _, arrow := range []*int{line.StartArrow, line.EndArrow} {
			if arrow != nil && (*arrow < 0 || *arrow > 19) {
				c.warnings = append(c.warnings, fmt.Sprintf("object %s: unsupported arrow %d; using no arrow", obj.ID, *arrow))
			}
		}
		if line.InterpolationType != "" && line.InterpolationType != "linear" &&
			line.InterpolationType != "orthogonal" && line.InterpolationType != "quadratic" {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s: unsupported line interpolation %q; using linear", obj.ID, line.InterpolationType))
		}
		styleParts = append(styleParts, "shape=filledEdge;",
			"strokeWidth="+format(math.Floor(line.StrokeWidth+0.5))+";",
			"strokeColor="+nonempty(line.StrokeColor, "#000000")+";",
			"fillColor="+nonempty(line.FillColor, "none")+";",
			arrowStyle(line.StartArrow, true),
			arrowStyle(line.EndArrow, false),
			"rounded="+boolInt(line.CornerRadius != nil)+";",
			dashStyle(line.DashStyle, math.Floor(line.StrokeWidth+0.5)),
			lineStyle(line.InterpolationType),
		)
		geometry.X, geometry.Y, geometry.Relative = "0", "0", "1"
	case g.Type == "Text" && g.Text != nil:
		vertex = true
		styleParts = append(styleParts, "text;html=1;nl2Br=0;")
		if obj.parent != nil && !obj.parent.isGroup() {
			if obj.parent.isLine() {
				geometry = lineLabelGeometry(obj)
				g.Text.halign = ""
				styleParts = append(styleParts, "labelBackgroundColor="+c.diagram.Stage.Background+";")
			} else {
				geometry.X, geometry.Y = "0", "0"
				geometry.Width, geometry.Height = obj.parent.cell.Geometry.Width, obj.parent.cell.Geometry.Height
			}
			geometry.Relative = "1"
		}
	case g.Type == "Image" && g.Image != nil:
		vertex = true
		styleParts = append(styleParts, "shape="+nonempty(c.translate(obj.UID, ""), "image")+";image="+strings.ReplaceAll(g.Image.URL, ";base64", "")+";")
	case g.Type == "Svg" && g.SVG != nil:
		vertex = true
		styleParts = append(styleParts, "shape=image;imageAspect=0;")
		if data := c.resourceData(g.SVG.EmbeddedResourceID); data != "" {
			if strings.Contains(data, "viewBox=") {
				c.warnings = append(c.warnings, fmt.Sprintf("object %s: SVG viewBox preserved; automatic artwork cropping requires an SVG renderer", obj.ID))
			}
			styleParts = append(styleParts, "image=data:image/svg+xml,"+base64.StdEncoding.EncodeToString([]byte(data))+";")
		}
	case g.Type == "Link":
		return "", "", nil, false, false
	default:
		return "", "", nil, false, false
	}
	style := strings.Join(styleParts, "")
	if !edge {
		style = rotateObject(obj, geometry, style)
	}
	if !obj.isGroup() {
		if text := findText(obj); text != nil {
			txt := text.Graphic.Text
			if obj.isSwimlane() {
				txt.forceTopPaddingShift, txt.Valign = true, "middle"
			}
			value = objectText(text)
			x, y := text.X, text.Y
			if shifts := obj.coordinateShifts(); len(shifts) == 4 {
				dx, dy, _, _ := obj.adjustShifts(shifts)
				x, y = x-dx, y-dy
			}
			if text == obj || strings.Contains(obj.UID, "chevron") {
				x, y = 0, 0
			}
			ts := textStyle(txt, x, y)
			if strings.Contains(obj.UID, "com.gliffy.shape.uml.uml_v2.activity.frame") {
				style = strings.ReplaceAll(style, "labelX=32", "labelX="+format(float64(float32(text.Width)*float32(1.1))))
				ts = strings.ReplaceAll(textStyle(txt, 0, 0), "verticalAlign=middle", "verticalAlign=top")
			}
			style += "html=1;nl2Br=0;"
			if !edge {
				style += ts
			} else {
				value = ""
			}
		}
	}
	style += "gliffyId=" + obj.ID + ";"
	return style, value, geometry, vertex, edge
}

func (o *Object) hasChildren() bool { return o != nil && len(o.Children) > 0 }

func isLink(o *Object) bool { return o != nil && o.Graphic != nil && o.Graphic.Type == "Link" }

func isPopupNote(o *Object) bool {
	return o != nil && o.Graphic != nil && o.Graphic.Type == "PopupNote"
}

func findText(o *Object) *Object {
	if o.Graphic != nil && o.Graphic.Type == "Text" && o.Graphic.Text != nil {
		return o
	}
	for _, child := range o.Children {
		if child == nil {
			continue
		}
		if child.Graphic != nil && child.Graphic.Type == "Text" && child.Graphic.Text != nil {
			return child
		}
		if found := findText(child); found != nil {
			found.X += child.X
			found.Y += child.Y
			return found
		}
	}
	return nil
}

func objectText(textObject *Object) string {
	if textObject.Graphic == nil || textObject.Graphic.Text == nil {
		return ""
	}
	width := textObject.Width
	if textObject.Graphic.Text.Overflow == "none" {
		width -= 3
	} else {
		width += 6
	}
	return fmt.Sprintf("<div style='width: %spx;height:auto;word-break: break-word;'>%s</div>", javaDecimal(width), textObject.Graphic.Text.HTML)
}

var alignmentPattern = regexp.MustCompile(`text-align: ?(left|center|right);`)
var alignmentStylePattern = regexp.MustCompile(`style="text-align:\s?(left|center|right);"`)
var spanPattern = regexp.MustCompile(`<span style="([^"]*)">`)

func writeStrings(builder *strings.Builder, values ...string) {
	for _, value := range values {
		builder.WriteString(value)
	}
}

func textHTML(value string) string {
	value = alignmentStylePattern.ReplaceAllString(value, `align="$1"`)
	var html strings.Builder
	last := 0
	for _, match := range spanPattern.FindAllStringSubmatchIndex(value, -1) {
		html.WriteString(value[last:match[3]])
		if !strings.Contains(value[match[2]:match[3]], "line-height:") {
			lineHeight := "normal"
			// Gliffy relies on quirks-mode sizing for spans containing another span.
			if strings.HasPrefix(strings.ToLower(value[match[1]:]), "<span") {
				lineHeight = "0"
			}
			writeStrings(&html, " line-height: ", lineHeight, ";")
		}
		last = match[3]
	}
	html.WriteString(value[last:])
	value = html.String()
	value = strings.ReplaceAll(value, "<p ", "<div ")
	value = strings.ReplaceAll(value, "<p>", "<div>")
	value = strings.ReplaceAll(value, "</p>", "</div>")
	return value
}

func textStyle(text *Text, x, y float64) string {
	var style strings.Builder
	switch text.VPosition {
	case "above":
		style.WriteString("verticalLabelPosition=top;verticalAlign=bottom;")
	case "below":
		style.WriteString("verticalLabelPosition=bottom;verticalAlign=top;")
	case "none":
		writeStrings(&style, "verticalAlign=", nonempty(text.Valign, "middle"), ";")
	}
	switch text.HPosition {
	case "left":
		style.WriteString("labelPosition=left;align=right;")
	case "right":
		style.WriteString("labelPosition=right;align=left;")
	case "none", "":
		writeStrings(&style, "align=", nonempty(text.halign, "center"), ";")
		if text.halign == "right" {
			x = 0
		}
	}
	left, right := numeric(text.PaddingLeft), numeric(text.PaddingRight)
	top, bottom := numeric(text.PaddingTop), numeric(text.PaddingBottom)
	writeStrings(&style, "spacingLeft=", format(math.Max(0, left-2)+x), ";")
	writeStrings(&style, "spacingRight=", format(math.Max(0, right-2)), ";")
	if text.forceTopPaddingShift || text.Valign != "middle" {
		writeStrings(&style, "spacingTop=", format(top-7+y), ";spacingBottom=", format(bottom), ";")
	}
	if text.Overflow == "none" {
		style.WriteString("whiteSpace=wrap;")
	}
	return style.String()
}

func javaDecimal(v float64) string {
	s := format(v)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func (c *converter) resourceData(id *int) string {
	if id == nil {
		c.warnings = append(c.warnings, "embedded SVG resource ID is missing")
		return ""
	}
	for _, resource := range c.diagram.EmbeddedResources.Resources {
		if resource.ID == *id {
			if resource.Data == "" {
				c.warnings = append(c.warnings, fmt.Sprintf("embedded SVG resource %d is empty", *id))
			}
			return resource.Data
		}
	}
	c.warnings = append(c.warnings, fmt.Sprintf("embedded SVG resource %d was not found", *id))
	return ""
}

func (c *converter) translate(uid, tid string) string {
	if shape, ok := c.translations[uid]; ok {
		return shape
	}
	if shape, ok := c.translations[tid]; ok {
		return shape
	}
	return ""
}

func arrowStyle(id *int, start bool) string {
	names := map[int]struct {
		name string
		fill bool
		size int
	}{
		0: {"none", false, 6}, 1: {"open", false, 6}, 2: {"block", true, 6},
		3: {"block", false, 6}, 4: {"block", false, 10}, 5: {"diamond", false, 12},
		6: {"open", false, 10}, 7: {"diamond", true, 12}, 8: {"classic", true, 6},
		9: {"ERzeroToMany", true, 10}, 10: {"ERoneToMany", true, 10},
		11: {"ERmandOne", true, 10}, 12: {"ERone", true, 10},
		13: {"ERzeroToOne", true, 10}, 14: {"ERmany", true, 10},
		15: {"oval", false, 10}, 16: {"dash", false, 6}, 17: {"block", true, 6},
		18: {"classic", true, 6}, 19: {"openAsync", false, 10},
	}
	value := struct {
		name string
		fill bool
		size int
	}{"none", false, 6}
	if id != nil {
		if mapped, ok := names[*id]; ok {
			value = mapped
		}
	}
	prefix := "end"
	if start {
		prefix = "start"
	}
	fill := "0"
	if value.fill {
		fill = "1"
	}
	style := fmt.Sprintf("%sArrow=%s;%sFill=%s;%sSize=%d;", prefix, value.name, prefix, fill, prefix, value.size)
	if id != nil && *id == 15 {
		terminal := "target"
		if start {
			terminal = "source"
		}
		style += terminal + "PerimeterSpacing=6;"
	}
	return style
}

func dashStyle(value string, width float64) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	for i, part := range parts {
		if number, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
			parts[i] = format(number * width)
		}
	}
	return "dashed=1;fixDash=1;dashPattern=" + strings.Join(parts, " ") + ";"
}

func lineStyle(value string) string {
	switch value {
	case "orthogonal":
		return "edgeStyle=orthogonal;"
	case "quadratic":
		return "curved=1;edgeStyle=orthogonalEdgeStyle;"
	default:
		return ""
	}
}

func numeric(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func nonempty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func boolInt(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func boolString(value bool) string { return boolInt(value) }

func format(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func (c MxCell) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	if c.UserObject != nil {
		wrapper := xml.StartElement{Name: xml.Name{Local: "UserObject"},
			Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: c.ID}}}
		for _, attr := range []struct{ key, value string }{
			{"link", c.UserObject.Link}, {"label", c.UserObject.Label}, {"tooltip", c.UserObject.Tooltip},
		} {
			if attr.value != "" {
				wrapper.Attr = append(wrapper.Attr, xml.Attr{Name: xml.Name{Local: attr.key}, Value: attr.value})
			}
		}
		if err := encoder.EncodeToken(wrapper); err != nil {
			return err
		}
		c.UserObject, c.ID, c.Value, c.HasValue = nil, "", "", false
		if err := c.MarshalXML(encoder, xml.StartElement{}); err != nil {
			return err
		}
		return encoder.EncodeToken(wrapper.End())
	}
	start.Name.Local = "mxCell"
	if c.ID != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: c.ID})
	}
	if c.Parent != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "parent"}, Value: c.Parent})
	}
	if c.Style != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "style"}, Value: c.Style})
	}
	if c.HasValue || c.Value != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "value"}, Value: c.Value})
	}
	if c.HasVertex {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "vertex"}, Value: c.Vertex})
	}
	if c.HasEdge {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "edge"}, Value: c.Edge})
	}
	if c.HasVisible {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "visible"}, Value: c.Visible})
	}
	start.Attr = append(start.Attr, c.Extra...)
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	if c.Geometry != nil {
		if err := encoder.EncodeElement(c.Geometry, xml.StartElement{Name: xml.Name{Local: "mxGeometry"}}); err != nil {
			return err
		}
	}
	return encoder.EncodeToken(start.End())
}

func (g MxGeometry) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "mxGeometry"
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "as"}, Value: g.As},
	}
	for _, attr := range []struct{ name, value string }{
		{"x", g.X}, {"y", g.Y}, {"width", g.Width}, {"height", g.Height}, {"relative", g.Relative},
	} {
		if attr.value != "" {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: attr.name}, Value: attr.value})
		}
	}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	if g.Points != nil {
		if err := encoder.EncodeElement(g.Points, xml.StartElement{Name: xml.Name{Local: "Array"}}); err != nil {
			return err
		}
	}
	for _, point := range []*MxPoint{g.SourcePoint, g.TargetPoint, g.Offset} {
		if point != nil {
			if err := encoder.EncodeElement(point, xml.StartElement{Name: xml.Name{Local: "mxPoint"}}); err != nil {
				return err
			}
		}
	}
	return encoder.EncodeToken(start.End())
}

func (p MxPointArray) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "Array"
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "as"}, Value: p.As}}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	for _, point := range p.Points {
		if err := encoder.EncodeElement(point, xml.StartElement{Name: xml.Name{Local: "mxPoint"}}); err != nil {
			return err
		}
	}
	return encoder.EncodeToken(start.End())
}

func (p MxPoint) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "mxPoint"
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "x"}, Value: p.X}, {Name: xml.Name{Local: "y"}, Value: p.Y}}
	if p.As != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "as"}, Value: p.As})
	}
	return encoder.EncodeElement(struct{}{}, start)
}
