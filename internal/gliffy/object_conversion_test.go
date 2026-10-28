package gliffy

import (
	"reflect"
	"testing"
)

func TestConvertObject(t *testing.T) {
	tests := []struct {
		name      string
		converter *converter
		object    *Object
		want      objectConversion
	}{
		{
			name:      "shape",
			converter: &converter{translations: map[string]string{}},
			object: &Object{
				ID: "shape-1", X: 10, Y: 20, Width: 30, Height: 40,
				Graphic: &Graphic{Type: GraphicTypeShape, Shape: &Shape{
					StrokeColor: "#000000", FillColor: "white", StrokeWidth: 1, Opacity: 1,
				}},
			},
			want: objectConversion{
				style:    "shadow=0;strokeWidth=1;fillColor=white;strokeColor=#000000;opacity=100;gliffyId=shape-1;",
				geometry: &MxGeometry{As: "geometry", X: "10", Y: "20", Width: "30", Height: "40"},
				vertex:   true,
			},
		},
		{
			name:      "line",
			converter: &converter{translations: map[string]string{}},
			object: &Object{
				ID: "line-1", X: 10, Y: 20, Width: 30, Height: 40,
				Graphic: &Graphic{Type: GraphicTypeLine, Line: &Line{StrokeWidth: 1}},
			},
			want: objectConversion{
				style:    "shape=filledEdge;strokeWidth=1;strokeColor=#000000;fillColor=none;startArrow=none;startFill=0;startSize=6;endArrow=none;endFill=0;endSize=6;rounded=0;gliffyId=line-1;",
				geometry: &MxGeometry{As: "geometry", X: "0", Y: "0", Width: "30", Height: "40", Relative: "1"},
				edge:     true,
			},
		},
		{
			name:      "text",
			converter: &converter{translations: map[string]string{}},
			object: &Object{
				ID: "text-1", Width: 10,
				Graphic: &Graphic{Type: GraphicTypeText, Text: &Text{HTML: "label"}},
			},
			want: objectConversion{
				style:    "text;html=1;nl2Br=0;html=1;nl2Br=0;align=center;spacingLeft=0;spacingRight=0;spacingTop=-7;spacingBottom=0;gliffyId=text-1;",
				value:    "<div style='width: 16.0px;height:auto;word-break: break-word;'>label</div>",
				geometry: &MxGeometry{As: "geometry", X: "0", Y: "0", Width: "10", Height: "0"},
				vertex:   true,
			},
		},
		{
			name:      "image",
			converter: &converter{translations: map[string]string{}},
			object: &Object{
				ID:      "image-1",
				Graphic: &Graphic{Type: GraphicTypeImage, Image: &Image{URL: "image.png;base64"}},
			},
			want: objectConversion{
				style:    "shape=image;image=image.png;gliffyId=image-1;",
				geometry: &MxGeometry{As: "geometry", X: "0", Y: "0", Width: "0", Height: "0"},
				vertex:   true,
			},
		},
		{
			name: "SVG",
			converter: &converter{
				translations: map[string]string{},
				resources: indexResources([]Resource{
					{ID: 1, Data: "<svg/>"},
				}),
			},
			object: &Object{
				ID:      "svg-1",
				Graphic: &Graphic{Type: GraphicTypeSVG, SVG: &SVG{EmbeddedResourceID: new(1)}},
			},
			want: objectConversion{
				style:    "shape=image;imageAspect=0;image=data:image/svg+xml,PHN2Zy8+;gliffyId=svg-1;",
				geometry: &MxGeometry{As: "geometry", X: "0", Y: "0", Width: "0", Height: "0"},
				vertex:   true,
			},
		},
		{
			name:      "unsupported graphic",
			converter: &converter{translations: map[string]string{}},
			object:    &Object{Graphic: &Graphic{Type: GraphicType("Unknown")}},
			want:      objectConversion{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.converter.convertObject(tt.object)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertObject() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
