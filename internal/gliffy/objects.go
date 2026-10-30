package gliffy

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

func shapeSet(prefix, names string) map[string]bool {
	result := make(map[string]bool)
	for name := range strings.FieldsSeq(names) {
		result[prefix+name] = true
	}
	return result
}

var groupShapes = shapeSet("com.gliffy.shape.", `
basic.basic_v1.default.group
erd.erd_v1.default.entity_with_attributes erd.erd_v1.default.entity_with_multiple_attributes
uml.uml_v2.sequence.interaction_use uml.uml_v2.sequence.opt_combined_fragment
uml.uml_v2.sequence.loop_combined_fragment uml.uml_v2.sequence.alt_combined_fragment
uml.uml_v2.class.object uml.uml_v2.class.enumeration uml.uml_v2.class.interface
uml.uml_v2.class.class2 uml.uml_v2.class.class uml.uml_v2.class.data_type
uml.uml_v2.state_machine.composite_state uml.uml_v2.state_machine.orthoganal_state
uml.uml_v2.class.package uml.uml_v2.sequence.boundary_lifeline uml.uml_v2.sequence.lifeline
uml.uml_v2.sequence.entity_lifeline uml.uml_v2.sequence.control_lifeline
uml.uml_v1.default.object_timeline uml.uml_v1.default.class uml.uml_v1.default.object
iphone.iphone_ios7.containers_content.table iphone.iphone_ios7.forms_controls.button_stack
iphone.iphone_ios7.forms_controls.alert_2options iphone.iphone_ios7.forms_controls.alert
iphone.iphone_ios7.navigation.contextual_menu iphone.iphone_ios7.navigation.nav_3tabs
iphone.iphone_ios7.containers_content.title_bar iphone.iphone_ios7.navigation.tab_bar
iphone.iphone_ios7.forms_controls.search android.android_v1.general.dialog
android.android_v1.general.list_1line android.android_v1.general.list_2lines
android.android_v1.general.tabs01 android.android_v1.general.tabs02
network.network_v3.business.user_group ui.ui_v3.navigation.navbar
ui.ui_v3.navigation.navbar_vertical ui.ui_v3.forms_controls.dropdown
uml.uml_v2.sequence.recursive_message`)

var graphiclessShapes = shapeSet("com.gliffy.shape.", `
erd.erd_v1.default.entity_with_attributes erd.erd_v1.default.entity_with_multiple_attributes
bpmn.bpmn_v1.data_artifacts.annotation
ui.ui_v3.navigation.navbar ui.ui_v3.forms_controls.combo_box
ui.ui_v3.containers_content.tooltip_top ui.ui_v3.containers_content.tooltip_bottom
ui.ui_v3.containers_content.tooltip_left ui.ui_v3.containers_content.tooltip_right
ui.ui_v3.containers_content.popover_top ui.ui_v3.containers_content.popover_bottom
ui.ui_v3.forms_controls.selector ui.ui_v3.icon_symbols.annotate_left
ui.ui_v3.icon_symbols.annotate_right ui.ui_v3.icon_symbols.annotate_top
ui.ui_v3.containers_content.speech_bubble_right ui.ui_v3.containers_content.speech_bubble_left`)

var sitemapShapes = shapeSet("com.gliffy.shape.sitemap.sitemap_v2.", `
page home gliffy form shopping_cart text video upload slideshow sitemap settings search
script print pricing photo map login game gallery download document chat calendar audio
profile error success cloud`)

func (o *Object) isGroup() bool {
	return o.hasChildren() && (groupShapes[o.UID] || strings.HasPrefix(o.UID, "com.gliffy.shape.table") ||
		(o.UID == "" && !o.Children[0].isText()))
}

func (o *Object) isText() bool {
	return o != nil && o.Graphic != nil && o.Graphic.Type == GraphicTypeText
}

func (o *Object) isLine() bool {
	return o != nil && o.Graphic != nil && o.Graphic.Type == GraphicTypeLine
}

func (o *Object) isSwimlane() bool {
	return strings.Contains(o.UID, "com.gliffy.shape.swimlanes")
}

func (o *Object) isSelection() bool {
	return strings.Contains(o.UID, "default.selection")
}

func (o *Object) isMindmap() bool {
	switch o.UID {
	case "com.gliffy.shape.mindmap.mindmap_v1.default.main_topic",
		"com.gliffy.shape.mindmap.mindmap_v1.default.subtopic",
		"com.gliffy.shape.mindmap.mindmap_v1.default.child_node":
		return true
	}
	return false
}

func (o *Object) effectiveGraphic() *Graphic {
	if o.Graphic != nil {
		return o.Graphic
	}
	if o.hasChildren() && o.Children[0] != nil && (strings.HasPrefix(o.UID, "com.gliffy.shape.uml.uml") ||
		graphiclessShapes[o.UID] || sitemapShapes[o.UID]) {
		return o.Children[0].Graphic
	}
	return nil
}

func (o *Object) normalizeGroup() {
	minX, minY := 0.0, 0.0
	for _, child := range o.Children {
		if child != nil {
			minX, minY = math.Min(minX, child.X), math.Min(minY, child.Y)
		}
	}
	o.X, o.Y, o.Width, o.Height = o.X+minX, o.Y+minY, o.Width-minX, o.Height-minY
	var maxOrder *int
	for _, child := range o.Children {
		if child == nil {
			continue
		}
		child.X, child.Y = child.X-minX, child.Y-minY
		if child.UID == "com.gliffy.shape.basic.basic_v1.default.rectangle" &&
			child.X == 0 && child.Y == 0 && child.Width == o.Width && child.Height == o.Height {
			if order, err := strconv.Atoi(rawOrder(child.Order)); err == nil && (maxOrder == nil || order > *maxOrder) {
				maxOrder = &order
			}
		}
	}
	if maxOrder != nil {
		o.Order = json.RawMessage(strconv.Itoa(*maxOrder))
	}
}

func shapeStyle(obj *Object, shape *Shape, translated string) string {
	var style strings.Builder
	appendDefault := func(key, value string) {
		if !strings.Contains(translated+style.String(), key) {
			writeStrings(&style, key, "=", value, ";")
		}
	}
	chevron := strings.Contains(obj.UID, "chevron")
	noFill := strings.Contains(shape.TID, "no_fill")
	appendDefault("shadow", boolString(shape.DropShadow))
	if !strings.Contains(translated, "strokeWidth") {
		writeStrings(&style, "strokeWidth=", format(math.Floor(shape.StrokeWidth+0.5)), ";")
		if math.Floor(shape.StrokeWidth+0.5) == 0 && !chevron {
			writeStrings(&style, "strokeColor=none", ";")
		}
	}
	if !strings.Contains(translated, "fillColor") {
		fill := shape.FillColor
		if noFill && !chevron {
			fill = "none"
		}
		writeStrings(&style, "fillColor=", fill, ";")
		if shape.FillColor == "none" {
			writeStrings(&style, "pointerEvents=0", ";")
		}
	}
	if !noFill {
		stroke := shape.StrokeColor
		if obj.UID == "com.gliffy.stencil.rectangle.no_fill_line_bottom_2px_off" ||
			(obj.UID == "" && shape.TID == "com.gliffy.stencil.rectangle.no_fill_line_bottom_2px_off") {
			stroke = shape.FillColor
		}
		appendDefault("strokeColor", stroke)
	}
	if !strings.Contains(translated, "gradient") && shape.Gradient &&
		!strings.HasPrefix(obj.UID, "com.gliffy.shape.venn.outline") &&
		!strings.HasPrefix(obj.UID, "com.gliffy.shape.venn.flat") {
		writeStrings(&style, "gradientColor=", gradientColor(obj, shape), ";gradientDirection=north;")
	}
	if !strings.HasPrefix(obj.UID, "com.gliffy.shape.venn") {
		appendDefault("opacity", format(shape.Opacity*100))
	}
	return style.String()
}

func gradientColor(obj *Object, shape *Shape) string {
	if obj.Graphic == nil || obj.Graphic.Shape == nil || obj.UID == "" ||
		strings.HasPrefix(obj.UID, "com.gliffy.shape.radial") || len(shape.FillColor) != 7 ||
		shape.FillColor[0] != '#' {
		return "#FFFFFF"
	}
	color, err := strconv.ParseUint(shape.FillColor[1:], 16, 32)
	if err != nil {
		return "#FFFFFF"
	}
	r := min(uint64(255), ((color>>16)&255)+170)
	g := min(uint64(255), ((color>>8)&255)+170)
	b := min(uint64(255), (color&255)+170)
	return fmt.Sprintf("#%06X", r<<16|g<<8|b)
}

func fragmentText(uid string) string {
	switch uid {
	case "com.gliffy.shape.uml.uml_v2.sequence.interaction_use":
		return "ref"
	case "com.gliffy.shape.uml.uml_v2.sequence.opt_combined_fragment":
		return "opt"
	case "com.gliffy.shape.uml.uml_v2.sequence.loop_combined_fragment":
		return "loop"
	case "com.gliffy.shape.uml.uml_v2.sequence.alt_combined_fragment":
		return "alt"
	}
	return ""
}

var lightboxPagePattern = regexp.MustCompile(`pageId=([^&]*)&`)
var lightboxNamePattern = regexp.MustCompile(`name=([^&]*)&`)

func (c *converter) decorate(obj *Object) {
	var link, tooltip string
	if obj.effectiveGraphic() != nil {
		link = objectLink(obj, strings.Contains(obj.UID, "com.gliffy.shape.basic.basic_v1.default.chevron_"))
	}
	for _, child := range obj.Children {
		if isPopupNote(child) && child.Graphic.PopupNote != nil {
			tooltip = child.Graphic.PopupNote.Text
			break
		}
	}
	if link == "" && tooltip == "" {
		return
	}
	page, name := lightboxPagePattern.FindStringSubmatch(link), lightboxNamePattern.FindStringSubmatch(link)
	if len(page) > 0 && len(name) > 0 {
		if _, err := strconv.ParseInt(page[1], 10, 64); err == nil {
			link = "/plugins/drawio/lightbox.action?ceoId=" + page[1] + "&diagramName=" + name[1] + ".drawio"
		} else {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s: invalid lightbox pageId %q; preserving link", obj.ID, page[1]))
		}
	}
	obj.cell.UserObject = &MxUserObject{Link: link, Tooltip: tooltip}
	if link != "" && !obj.isGroup() {
		obj.cell.UserObject.Label = obj.cell.Value
	}
}

func objectLink(obj *Object, deep bool) string {
	for _, child := range obj.Children {
		if isLink(child) && child.Graphic.Link != nil {
			return child.Graphic.Link.Href
		}
		if deep && child != nil {
			if link := objectLink(child, true); link != "" {
				return link
			}
		}
	}
	return ""
}

func (c *converter) swimlaneStyle(obj *Object) string {
	header := obj.Children[0]
	if header == nil || header.Graphic == nil || header.Graphic.Shape == nil {
		c.warnings = append(c.warnings, fmt.Sprintf("object %s: swimlane header has no shape", obj.ID))
		return "swimlane;"
	}
	shape := header.Graphic.Shape
	style := c.translate(obj.UID, "") + ";"
	if obj.Rotation == 0 {
		style += "childLayout=stackLayout;resizeParent=1;resizeParentMax=0;"
	}
	return style + laneShapeStyle(shape) + "startSize=" + format(header.Height) + ";whiteSpace=wrap;"
}

func laneShapeStyle(shape *Shape) string {
	return "strokeWidth=" + format(math.Floor(shape.StrokeWidth+0.5)) + ";shadow=" + boolString(shape.DropShadow) +
		";fillColor=" + shape.FillColor + ";strokeColor=" + shape.StrokeColor + ";"
}

func (c *converter) convertLanes(obj *Object) {
	for _, lane := range obj.Children[1:] {
		if lane == nil || lane.Graphic == nil || lane.Graphic.Shape == nil || !lane.hasChildren() ||
			!lane.Children[0].isText() || lane.Children[0].Graphic.Text == nil {
			c.warnings = append(c.warnings, fmt.Sprintf("object %s: malformed swimlane lane", obj.ID))
			continue
		}
		lane.parent = obj
		style := "swimlane;collapsible=0;swimlaneLine=0;" + laneShapeStyle(lane.Graphic.Shape) +
			"whiteSpace=wrap;html=1;fontStyle=0;"
		x, y, w, h := lane.X, lane.Y, lane.Width, lane.Height
		if obj.Rotation == 270 {
			style += "horizontal=0;"
			x, y, w, h = y, obj.Width-w-x, h, w
		} else if obj.Rotation != 0 {
			style += "rotation=" + format(obj.Rotation) + ";"
			x, y = rotatedGeometry(x, y, w, h, obj.Rotation, obj.Width/2, obj.Height/2)
		}
		text := lane.Children[0]
		style += textStyle(text.Graphic.Text, 0, 0) + "gliffyId=" + lane.ID + ";"
		lane.cell = &MxCell{Style: style, Value: objectText(text), HasValue: true,
			Vertex: "1", HasVertex: true, Geometry: &MxGeometry{As: "geometry",
				X: format(x), Y: format(y), Width: format(w), Height: format(h)}}
		obj.importChildren = append(obj.importChildren, lane)
	}
	if obj.Rotation != 0 {
		for i, j := 0, len(obj.importChildren)-1; i < j; i, j = i+1, j-1 {
			obj.importChildren[i], obj.importChildren[j] = obj.importChildren[j], obj.importChildren[i]
		}
	}
}
