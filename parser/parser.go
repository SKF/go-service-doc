package parser

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/Depado/bfchroma"
	"github.com/pkg/errors"
	"github.com/russross/blackfriday/v2"
	"go.uber.org/zap"

	"github.com/SKF/go-service-doc/core"
	html_gen "github.com/SKF/go-service-doc/html-gen"
	"github.com/SKF/go-service-doc/utils"
)

type Parser struct {
	sourceDir       string
	outputDir       string
	basepath        string
	serviceFilename string
	serviceName     string
	serviceTitle    string
	uniqueLinks     map[string]bool
	pages           core.Pages
	staticFiles     core.Files
	searchPage      string
	faviconHref     string
	err             error
}

func NewParser() *Parser {
	p := Parser{
		uniqueLinks: make(map[string]bool),
	}

	return &p
}

func (se *Parser) WithSourceDir(sourceDir string) *Parser {
	se.sourceDir = sourceDir
	return se
}

func (se *Parser) WithOutputDir(outputDir string) *Parser {
	se.outputDir = outputDir
	return se
}

func (se *Parser) WithBasepath(basepath string) *Parser {
	se.basepath = basepath
	return se
}

func (se *Parser) ServiceFilename(serviceFilename string) *Parser {
	se.serviceFilename = serviceFilename
	return se
}

// nolint: stylecheck
func (p *Parser) Error() error {
	return p.err
}

func (p *Parser) Pages() core.Pages {
	return p.pages
}

func (p *Parser) StaticFiles() core.Files {
	return p.staticFiles
}

func (p *Parser) SearchPage() string {
	return p.searchPage
}

func (p *Parser) Run() {
	p.findMDFiles()
	p.findStaticFiles()
	p.parseMarkdown()
	p.enrichIndexDocumentsWithHTML()
	p.buildHTMLPages()
	p.buildSearchPage()
}

func (p *Parser) findMDFiles() {
	zap.L().Info("search for MD filesz")

	files, err := ioutil.ReadDir(p.sourceDir + "/")
	if err != nil {
		p.err = errors.Wrap(err, "ioutil.ReadDir failed")
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}

		page := core.Page{}
		page.Name = strings.ReplaceAll(f.Name(), ".md", "")
		page.Name = utils.ConvertToCamelCase(page.Name)

		if f.Name() != p.serviceFilename {
			page.WebPath = p.basepath + "/" + utils.ConvertToKebabCase(page.Name)
		} else {
			page.WebPath = p.basepath
			p.serviceName = page.Name
		}

		page.Filepath = p.sourceDir + "/" + f.Name()
		p.pages = append(p.pages, page)
	}
}

func (p *Parser) parseMarkdown() {
	for idx, pg := range p.pages {
		page := pg
		zap.L().With(zap.String("page", page.Name)).Info("parsing markdown")

		content, err := ioutil.ReadFile(page.Filepath)
		if err != nil {
			p.err = errors.Wrap(err, "ioutil.ReadFile failed")
			return
		}

		// Convert Markdown to HTML
		exts := blackfriday.NoIntraEmphasis |
			blackfriday.AutoHeadingIDs |
			blackfriday.HeadingIDs |
			blackfriday.FencedCode |
			blackfriday.HardLineBreak |
			blackfriday.Tables

		markdown := blackfriday.Run(
			content,
			blackfriday.WithRenderer(bfchroma.NewRenderer()),
			blackfriday.WithExtensions(exts),
		)
		page.Markdown = string(markdown)

		// Build Menu from Markdown
		menuNode := blackfriday.New(blackfriday.WithExtensions(blackfriday.HeadingIDs)).Parse(content)
		menuNode.Walk(p.menuWalker(&page))

		// Build Search Index Documents from Markdown
		searchNode := blackfriday.New(blackfriday.WithExtensions(blackfriday.AutoHeadingIDs | blackfriday.HeadingIDs)).Parse(content)
		searchNode.Walk(p.searchWalker(&page))

		p.pages[idx] = page
	}
}

func (p *Parser) menuWalker(page *core.Page) blackfriday.NodeVisitor {
	return func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if node.Type != blackfriday.Heading || !entering || string(node.FirstChild.Literal) == "" {
			return blackfriday.GoToNext
		}

		if node.Level > 2 || node.HeadingID == "" {
			if node.HeadingID != "" {
				return blackfriday.GoToNext
			}

			return blackfriday.GoToNext
		}

		link := fmt.Sprintf("%s#%s", page.WebPath, node.HeadingID)
		if exists := p.uniqueLinks[link]; exists {
			p.err = errors.Errorf("link already exists, [%s]", link)
		}

		p.uniqueLinks[link] = true
		h := core.Header{Title: string(node.FirstChild.Literal), Link: link}

		if node.Level == 1 {
			if p.serviceName == page.Name && p.serviceTitle == "" {
				p.serviceTitle = h.Title
			}

			page.Headers = append(page.Headers, h)

			return blackfriday.GoToNext
		}

		idx := len(page.Headers) - 1
		page.Headers[idx].Headers = append(page.Headers[idx].Headers, h)

		return blackfriday.GoToNext
	}
}

func (p *Parser) searchWalker(page *core.Page) blackfriday.NodeVisitor {
	var (
		currentDoc    core.IndexDocument
		uniqueIndexes = map[string]bool{}
	)

	return func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if !entering {
			if node.Type == blackfriday.Document {
				page.IndexDocuments = append(page.IndexDocuments, currentDoc)
			}

			return blackfriday.GoToNext
		}

		if node.Type == blackfriday.Heading {
			if node.Level != 1 {
				page.IndexDocuments = append(page.IndexDocuments, currentDoc)
			}

			var ctxIdx = node.Level - 1
			if len(currentDoc.Context) < ctxIdx {
				ctxIdx = len(currentDoc.Context) - 1
			}

			var context = make([]string, len(currentDoc.Context[:ctxIdx])+1)

			copy(context[:ctxIdx], currentDoc.Context[:ctxIdx])
			context[len(context)-1] = string(node.FirstChild.Literal)

			originalHeadingID := node.HeadingID
			uniqueID := originalHeadingID
			n := 1

			for uniqueIndexes[uniqueID] {
				uniqueID = fmt.Sprintf("%s-%d", originalHeadingID, n)
				n++
			}

			uniqueIndexes[uniqueID] = true

			currentDoc = core.IndexDocument{}
			currentDoc.ID = uniqueID
			currentDoc.Link = fmt.Sprintf("%s#%s", page.WebPath, uniqueID)
			currentDoc.Context = context

			return blackfriday.GoToNext
		}

		content := string(node.Literal)
		content = strings.TrimSpace(content)

		content = strings.ReplaceAll(content, "`", "` + "+`"`+"`"+`"`+" + `")

		if content != "" {
			currentDoc.Content = append(currentDoc.Content, content)
		}

		return blackfriday.GoToNext
	}
}

func (p *Parser) enrichIndexDocumentsWithHTML() {
	for idx, page := range p.pages {
		for jdx, doc := range page.IndexDocuments {
			doc.Context = append([]string{p.serviceTitle}, doc.Context...)
			page.IndexDocuments[jdx] = doc
		}

		for jdx := 0; jdx < len(page.IndexDocuments); jdx++ {
			re1 := regexp.MustCompile(`\<h\d+ id\=\"` + page.IndexDocuments[jdx].ID + `\"`)
			idx1 := re1.FindStringIndex(page.Markdown)[0]

			var idx2 = len(page.Markdown)

			if jdx != len(page.IndexDocuments)-1 {
				re2 := regexp.MustCompile(`\<h\d+ id\=\"` + page.IndexDocuments[jdx+1].ID + `\"`)
				idx2 = re2.FindStringIndex(page.Markdown)[0]
			}

			html := page.Markdown[idx1:idx2]
			html = strings.TrimSpace(html)
			html = strings.ReplaceAll(html, "`", "` + "+`"`+"`"+`"`+" + `")

			page.IndexDocuments[jdx].HTML = html
		}

		p.pages[idx] = page
	}
}

func (p *Parser) buildSearchPage() {
	p.pages.SortByName(p.serviceName)

	searchPage, err := html_gen.New().
		WithAPITitle(p.serviceTitle).
		WithPages(p.pages).
		WithSearchLink("/search").
		WithBasepath(p.basepath).
		BuildSearchPageTemplate()
	if err != nil {
		p.err = errors.Wrap(err, "html_gen.BuildSearchPageTemplate failed")
		return
	}

	p.searchPage = string(searchPage)
}

func (p *Parser) buildHTMLPages() {
	p.pages.SortByName(p.serviceName)

	for idx, page := range p.pages {
		zap.L().With(zap.String("page", page.Name)).Info("building HTML page")

		bs, err := html_gen.New().
			WithAPITitle(p.serviceTitle).
			WithPages(p.pages).
			WithDocument(page.Markdown).
			WithSearchLink("/search").
			WithBasepath(p.basepath).
			WithFavicon(p.faviconHref).
			Build()
		if err != nil {
			p.err = errors.Wrap(err, "html_gen.Build failed")
			return
		}

		page.HTML = string(bs)
		p.pages[idx] = page
	}
}
