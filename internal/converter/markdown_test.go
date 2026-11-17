package converter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"conf2md/internal/config"
	"conf2md/internal/goconfluence"
	"conf2md/internal/mkdocs"
)

func TestConvertConfluenceToMarkdownBytes(t *testing.T) {
	got, err := convertConfluenceToMarkdown("<p>Hello <strong>world</strong>.</p>")
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("Hello **world**."); !bytes.Equal(got, want) {
		t.Errorf("Markdown = %q, want %q", got, want)
	}
}

func TestConvertConfluenceToMarkdownEscapesLeadingHash(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"paragraph", "<p>#heading</p>", `\#heading`},
		{"ordered list item", "<ol><li>#1 sample</li></ol>", "1. \\#1 sample\n"},
		{"unordered list item", "<ul><li>#tag</li></ul>", "- \\#tag\n"},
		{"heading content", "<h1>#title</h1>", "# \\#title"},
		{"inline hash", "<p>text #tag</p>", "text #tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertConfluenceToMarkdown(tt.html)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("Markdown = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertConfluenceToMarkdownPreservesUnderlineWithBold(t *testing.T) {
	got, err := convertConfluenceToMarkdown("<u><strong>right click</strong></u>")
	if err != nil {
		t.Fatal(err)
	}
	if want := "<ins>**right click**</ins>"; string(got) != want {
		t.Errorf("Markdown = %q, want %q", got, want)
	}
}

func TestConvertConfluenceToMarkdownListEndsWithNewline(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"ordered", "<ol><li>first</li><li>second</li></ol>", "1. first\n2. second\n"},
		{"unordered", "<ul><li>first</li><li>second</li></ul>", "- first\n- second\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertConfluenceToMarkdown(tt.html)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("Markdown = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertConfluenceToMarkdownOrderedListNumbering(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"start attribute", `<ol start="4"><li>first</li><li>second</li></ol>`, "4. first\n5. second\n"},
		{"item value", `<ol><li>first</li><li value="7">second</li><li>third</li></ol>`, "1. first\n7. second\n8. third\n"},
		{"reversed", `<ol reversed><li>first</li><li>second</li></ol>`, "2. first\n1. second\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertConfluenceToMarkdown(tt.html)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("Markdown = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertConfluenceToMarkdownAddsNewlineAfterEachList(t *testing.T) {
	got, err := convertConfluenceToMarkdown("<ul><li>outer<ul><li>inner</li></ul>after inner list</li></ul>after outer list<ol><li>last</li></ol>after ordered list")
	if err != nil {
		t.Fatal(err)
	}

	for _, text := range []string{"after inner list", "after outer list", "after ordered list"} {
		index := strings.Index(string(got), text)
		if index == -1 {
			t.Fatalf("Markdown %q does not contain %q", got, text)
		}
		prefix := bytes.TrimRight(got[:index], " \t")
		newlineCount := 0
		for i := len(prefix) - 1; i >= 0 && prefix[i] == '\n'; i-- {
			newlineCount++
		}
		if newlineCount != 2 {
			t.Errorf("Markdown has %d newlines before %q, want 2: %q", newlineCount, text, got)
		}
	}
}

func TestConvertConfluenceToMarkdownDoesNotAddNewlineAfterFollowingParagraph(t *testing.T) {
	got, err := convertConfluenceToMarkdown("<ul><li>item</li></ul><p>After</p>")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(string(got), "\n") {
		t.Errorf("Markdown = %q, should not have a final newline after a paragraph", got)
	}
}

func TestUniqueChildPageNames(t *testing.T) {
	children := []goconfluence.Results{
		{ID: "10", Title: "A Page"},
		{ID: "11", Title: "a-page"},
		{ID: "12", Title: "Unique"},
		{ID: "13", Title: "!!!"},
		{ID: "14", Title: "???"},
	}
	want := []string{"a-page-10", "a-page-11", "unique", "untitled-13", "untitled-14"}

	if got := uniqueChildPageNames(children); !reflect.DeepEqual(got, want) {
		t.Errorf("uniqueChildPageNames() = %v, want %v", got, want)
	}
}

func TestPageOutputLocationUsesNestedDirectory(t *testing.T) {
	content := &goconfluence.Content{Title: "Child Group"}
	children := &goconfluence.Search{Results: []goconfluence.Results{{ID: "20"}}}

	tests := []struct {
		name     string
		level    int
		children *goconfluence.Search
		pageName string
		wantDir  string
		wantFile string
	}{
		{"root", 0, children, "", "docs/parent", "index"},
		{"nested index", 2, children, "child-group", "docs/parent/child-group", "index"},
		{"nested leaf", 2, &goconfluence.Search{}, "child-20", "docs/parent", "child-20"},
		{"title fallback", 2, children, "", "docs/parent/child-group", "index"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, filename := pageOutputLocation(content, tt.children, tt.level, filepath.FromSlash("docs/parent"), tt.pageName)
			if dir != filepath.FromSlash(tt.wantDir) || filename != tt.wantFile {
				t.Errorf("pageOutputLocation() = (%q, %q), want (%q, %q)", dir, filename, tt.wantDir, tt.wantFile)
			}
		})
	}
}

func TestConvertAndWritePageKeepsNestedMkDocsPath(t *testing.T) {
	args := &config.CmdLineArgs{OutputDir: t.TempDir()}
	content := &goconfluence.Content{Title: "Child", Body: goconfluence.Body{
		Storage: goconfluence.Storage{Value: "<p>content</p>"},
	}}

	root := mkdocs.NewMkDocs("root", "root")
	page, err := newCrawler(nil, args, nil).convertAndWritePage(content, "docs/parent/child", "index", root)
	if err != nil {
		t.Fatalf("convertAndWritePage() error = %v", err)
	}

	if got, want := page.Path, "parent/child/index.md"; got != want {
		t.Errorf("MkDocs page path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(args.OutputDir, "docs", filepath.FromSlash(page.Path))); err != nil {
		t.Fatalf("generated page does not exist at its navigation path: %v", err)
	}
	navigation, err := root.GenerateMkDocs()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(navigation, "'Child': parent/child/index.md") {
		t.Errorf("generated MkDocs navigation lost nested path: %s", navigation)
	}
}

func TestCrawlConfluencePreservesParentAndChildMarkdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any
		switch r.URL.Path {
		case "/content/parent":
			response = goconfluence.Content{ID: "parent", Title: "Parent", Body: goconfluence.Body{
				Storage: goconfluence.Storage{Value: "<p>Parent content</p>"},
			}, Children: &goconfluence.Children{Page: &goconfluence.Search{
				Results: []goconfluence.Results{{ID: "child", Title: "Child"}},
			}}}
		case "/content/child":
			response = goconfluence.Content{ID: "child", Title: "Child", Body: goconfluence.Body{
				Storage: goconfluence.Storage{Value: "<p>Child content</p>"},
			}, Children: &goconfluence.Children{Page: &goconfluence.Search{}}}
		case "/content/parent/child/page":
			response = goconfluence.Search{Results: []goconfluence.Results{{ID: "child", Title: "Child"}}}
		case "/content/child/child/page", "/content/parent/child/attachment", "/content/child/child/attachment":
			response = goconfluence.Search{}
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()
	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	args := &config.CmdLineArgs{OutputDir: t.TempDir(), ExportChildren: true}
	root := mkdocs.NewMkDocs("root", "root")
	crawler := newCrawler(api, args, context.Background())
	var logs bytes.Buffer
	crawler.logger = zerolog.New(&logs).With().Str("component", "crawler").Logger()
	if err := crawler.crawlConfluence(0, "docs", "parent", "", root); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"component":"crawler"`, `"page_id":"parent"`, `"page_id":"child"`, `"depth":0`, `"depth":1`} {
		if !strings.Contains(logs.String(), field) {
			t.Errorf("crawler logs missing %s: %s", field, logs.String())
		}
	}
	for path, want := range map[string]string{
		"docs/index.md": "Parent content",
		"docs/child.md": "Child content",
	} {
		got, err := os.ReadFile(filepath.Join(args.OutputDir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if len(root.Children) != 1 || len(root.Children[0].Children) != 1 {
		t.Fatalf("unexpected exported navigation tree: %+v", root)
	}
}

func TestCrawlConfluencePreservesConfluenceChildPageOrderInMkDocs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any
		switch r.URL.Path {
		case "/content/parent":
			// API returns children out of sidebar order (e.g. creation/ID order).
			response = goconfluence.Content{
				ID:    "parent",
				Title: "Parent",
				Body:  goconfluence.Body{Storage: goconfluence.Storage{Value: "<p>Parent</p>"}},
				Children: &goconfluence.Children{Page: &goconfluence.Search{
					Limit: 25,
					Results: []goconfluence.Results{
						{ID: "3", Title: "Third Step", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`300`)}},
						{ID: "1", Title: "First Step", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`100`)}},
						{ID: "2", Title: "Second Step", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`200`)}},
					},
				}},
			}
		case "/content/1", "/content/2", "/content/3":
			id := strings.TrimPrefix(r.URL.Path, "/content/")
			response = goconfluence.Content{
				ID:       id,
				Title:    "Page " + id,
				Body:     goconfluence.Body{Storage: goconfluence.Storage{Value: "<p>" + id + "</p>"}},
				Children: &goconfluence.Children{Page: &goconfluence.Search{}},
			}
		case "/content/parent/child/attachment",
			"/content/1/child/attachment",
			"/content/2/child/attachment",
			"/content/3/child/attachment":
			response = goconfluence.Search{}
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	args := &config.CmdLineArgs{OutputDir: t.TempDir(), ExportChildren: true}
	root := mkdocs.NewMkDocs("root", "root")
	if err := newCrawler(api, args, context.Background()).crawlConfluence(0, "docs", "parent", "", root); err != nil {
		t.Fatal(err)
	}

	got := make([]string, len(root.Children[0].Children))
	for i, child := range root.Children[0].Children {
		got[i] = child.Path
	}
	want := []string{"first-step.md", "second-step.md", "third-step.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MkDocs child order = %v, want %v", got, want)
	}
}

func TestDownloadAttachmentsFetchesReferencedGliffyFromOtherPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any
		switch r.URL.Path {
		case "/content/current/child/attachment":
			response = goconfluence.Search{Results: []goconfluence.Results{{
				Title: "local.txt",
				Links: &goconfluence.Links{Download: "/download/local.txt"},
			}}}
		case "/content/remote/child/attachment":
			response = goconfluence.Search{Results: []goconfluence.Results{
				{Title: "diagram.gliffy", Links: &goconfluence.Links{Download: "/download/diagram.gliffy"}},
				{Title: "unreferenced.txt", Links: &goconfluence.Links{Download: "/download/unreferenced.txt"}},
			}}
		case "/download/local.txt":
			_, _ = w.Write([]byte("local attachment"))
			return
		case "/download/diagram.gliffy":
			_, _ = w.Write([]byte("referenced Gliffy attachment"))
			return
		case "/download/unreferenced.txt":
			t.Error("downloaded an unreferenced attachment from another page")
			http.NotFound(w, r)
			return
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	args := &config.CmdLineArgs{OutputDir: t.TempDir()}
	page := &goconfluence.Content{
		ID:    "current",
		Title: "Current",
		Body: goconfluence.Body{Storage: goconfluence.Storage{Value: `
			<ac:structured-macro ac:name="gliffy">
				<ac:parameter ac:name="pageid">remote</ac:parameter>
				<ac:parameter ac:name="name">diagram.gliffy</ac:parameter>
			</ac:structured-macro>
			<ac:structured-macro ac:name="gliffy">
				<ac:parameter ac:name="pageid">current</ac:parameter>
				<ac:parameter ac:name="name">same-page.gliffy</ac:parameter>
			</ac:structured-macro>
			<ac:structured-macro ac:name="other">
				<ac:parameter ac:name="pageid">ignored</ac:parameter>
				<ac:parameter ac:name="name">ignored.gliffy</ac:parameter>
			</ac:structured-macro>
		`}},
	}

	if err := newCrawler(api, args, context.Background()).downloadAttachments("docs", page); err != nil {
		t.Fatalf("downloadAttachments() error = %v", err)
	}

	for path, want := range map[string]string{
		"docs/attachments/local.txt":      "local attachment",
		"docs/attachments/diagram.gliffy": "referenced Gliffy attachment",
	} {
		got, err := os.ReadFile(filepath.Join(args.OutputDir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read downloaded attachment %q: %v", path, err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestCrawlConfluenceStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/content/parent":
			cancel()
			_ = json.NewEncoder(w).Encode(goconfluence.Content{ID: "parent", Title: "Parent",
				Children: &goconfluence.Children{Page: &goconfluence.Search{
					Results: []goconfluence.Results{{ID: "child", Title: "Child"}},
				}}})
		default:
			t.Errorf("unexpected request after cancellation: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	args := &config.CmdLineArgs{OutputDir: t.TempDir(), ExportChildren: true}
	err = newCrawler(api, args, ctx).crawlConfluence(0, "docs", "parent", "", mkdocs.NewMkDocs("root", "root"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("crawlConfluence() error = %v, want context.Canceled", err)
	}
}

func TestCrawlConfluenceWritesDebugPageJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any
		switch r.URL.Path {
		case "/content/page-123":
			response = goconfluence.Content{
				ID:       "page-123",
				Title:    "Debug page",
				Body:     goconfluence.Body{Storage: goconfluence.Storage{Value: "<p>Body</p>"}},
				Children: &goconfluence.Children{Page: &goconfluence.Search{}},
			}
		case "/content/page-123/child/attachment":
			response = goconfluence.Search{}
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	args := &config.CmdLineArgs{OutputDir: t.TempDir(), DebugLevel: "debug"}
	crawler := newCrawler(api, args, context.Background())
	if err := crawler.crawlConfluence(0, "docs", "page-123", "", mkdocs.NewMkDocs("root", "root")); err != nil {
		t.Fatalf("crawlConfluence() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(args.OutputDir, "json", "page-123.json"))
	if err != nil {
		t.Fatalf("read debug JSON: %v", err)
	}
	var got goconfluence.Content
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal debug JSON: %v", err)
	}
	if got.ID != "page-123" || got.Body.Storage.Value != "<p>Body</p>" {
		t.Errorf("debug JSON content = (%q, %q), want (%q, %q)", got.ID, got.Body.Storage.Value, "page-123", "<p>Body</p>")
	}
}

func TestSanitizeConfluenceHTMLRemovesEmptyParagraphs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "<p>a</p><p></p><p>b</p>", "<p>a</p><p>b</p>"},
		{"whitespace only", "<p>a</p><p> \n\t</p>", "<p>a</p>"},
		{"self-closing", "<p/><p />x", "x"},
		{"attributes", `<p class="auto-cursor-target"></p><P style="x" />y`, "y"},
		{"keeps content", "<p> a </p><p><br/></p><p>&nbsp;</p>", "<p> a </p><p><br/></p><p>&nbsp;</p>"},
		{"keeps other tags", "<pre></pre><param/><progress></progress>", "<pre></pre><param/><progress></progress>"},
		{
			"keeps code block content",
			"<p></p><ac:plain-text-body><![CDATA[<p></p>]]></ac:plain-text-body>",
			"<pre><code>&lt;p&gt;&lt;/p&gt;</code></pre>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeConfluenceHTML(tt.input); got != tt.want {
				t.Errorf("sanitizeConfluenceHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertConfluenceToMarkdownTextFormatting(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p><u>x</u></p>", "<ins>x</ins>"},
		{"<p><sub>x</sub></p>", "~x~"},
		{"<p><sup>x</sup></p>", "^x^"},
		{"<p><strong>x</strong></p>", "**x**"},
		{"<p><b>x</b></p>", "**x**"},
		{"<p><em>x</em></p>", "_x_"},
		{"<p><i>x</i></p>", "_x_"},
		{"<p><s>x</s></p>", "~~x~~"},
		{"<p><del>x</del></p>", "~~x~~"},
		{"<pre>x</pre>", "`x`"},
		{"<pre><code>x</code></pre>", "x"},
		{"<p><strong><em>x</em></strong></p>", "**_x_**"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := convertConfluenceToMarkdown(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("convertConfluenceToMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
