package md2jsV2

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"unicode"
	"unicode/utf8"
	"time"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/goccy/go-yaml"
)

type metaInp struct {
	Title string `yaml:"title"`
	Author string `yaml:"author"`
	Date time.Time `yaml:"date"`
}

func GetMeta(indata []byte) (meta *metaInp, err error) {

	var metaData metaInp
	err = yaml.Unmarshal(indata, &metaData)
	if err != nil {return nil, fmt.Errorf("unmarshall: %v", err)}

	return &metaData, nil
}


// A Config struct has configurations for the HTML based renderers.
type Config struct {
	Writer              Writer
	HardWraps           bool
	EastAsianLineBreaks EastAsianLineBreaks
	XHTML               bool
	Unsafe              bool
}

// NewConfig returns a new Config with defaults.
func NewConfig() Config {
	return Config{
		Writer:              DefaultWriter,
		HardWraps:           false,
		EastAsianLineBreaks: EastAsianLineBreaksNone,
		XHTML:               false,
		Unsafe:              false,
	}
}

// SetOption implements renderer.NodeRenderer.SetOption.
func (c *Config) SetOption(name renderer.OptionName, value interface{}) {
	switch name {
	case optHardWraps:
		c.HardWraps = value.(bool)
	case optEastAsianLineBreaks:
		c.EastAsianLineBreaks = value.(EastAsianLineBreaks)
	case optXHTML:
		c.XHTML = value.(bool)
	case optUnsafe:
		c.Unsafe = value.(bool)
	case optTextWriter:
		c.Writer = value.(Writer)
	}
}

// An Option interface sets options for HTML based renderers.
type Option interface {
	SetHTMLOption(*Config)
}

// TextWriter is an option name used in WithWriter.
const optTextWriter renderer.OptionName = "Writer"

type withWriter struct {
	value Writer
}

func (o *withWriter) SetConfig(c *renderer.Config) {
	c.Options[optTextWriter] = o.value
}

func (o *withWriter) SetHTMLOption(c *Config) {
	c.Writer = o.value
}

// WithWriter is a functional option that allow you to set the given writer to
// the renderer.
func WithWriter(writer Writer) interface {
	renderer.Option
	Option
} {
	return &withWriter{writer}
}

// HardWraps is an option name used in WithHardWraps.
const optHardWraps renderer.OptionName = "HardWraps"

type withHardWraps struct {
}

func (o *withHardWraps) SetConfig(c *renderer.Config) {
	c.Options[optHardWraps] = true
}

func (o *withHardWraps) SetHTMLOption(c *Config) {
	c.HardWraps = true
}

// WithHardWraps is a functional option that indicates whether softline breaks
// should be rendered as '<br>'.
func WithHardWraps() interface {
	renderer.Option
	Option
} {
	return &withHardWraps{}
}

// EastAsianLineBreaks is an option name used in WithEastAsianLineBreaks.
const optEastAsianLineBreaks renderer.OptionName = "EastAsianLineBreaks"

// A EastAsianLineBreaks is a style of east asian line breaks.
type EastAsianLineBreaks int

const (
	//EastAsianLineBreaksNone renders line breaks as it is.
	EastAsianLineBreaksNone EastAsianLineBreaks = iota
	// EastAsianLineBreaksSimple follows east_asian_line_breaks in Pandoc.
	EastAsianLineBreaksSimple
	// EastAsianLineBreaksCSS3Draft follows CSS text level3 "Segment Break Transformation Rules" with some enhancements.
	EastAsianLineBreaksCSS3Draft
)

func (b EastAsianLineBreaks) softLineBreak(thisLastRune rune, siblingFirstRune rune) bool {
	switch b {
	case EastAsianLineBreaksNone:
		return false
	case EastAsianLineBreaksSimple:
		return !(util.IsEastAsianWideRune(thisLastRune) && util.IsEastAsianWideRune(siblingFirstRune))
	case EastAsianLineBreaksCSS3Draft:
		return eastAsianLineBreaksCSS3DraftSoftLineBreak(thisLastRune, siblingFirstRune)
	}
	return false
}

func eastAsianLineBreaksCSS3DraftSoftLineBreak(thisLastRune rune, siblingFirstRune rune) bool {
	// Implements CSS text level3 Segment Break Transformation Rules with some enhancements.
	// References:
	//   - https://www.w3.org/TR/2020/WD-css-text-3-20200429/#line-break-transform
	//   - https://github.com/w3c/csswg-drafts/issues/5086

	// Rule1:
	//   If the character immediately before or immediately after the segment break is
	//   the zero-width space character (U+200B), then the break is removed, leaving behind the zero-width space.
	if thisLastRune == '\u200B' || siblingFirstRune == '\u200B' {
		return false
	}

	// Rule2:
	//   Otherwise, if the East Asian Width property of both the character before and after the segment break is
	//   F, W, or H (not A), and neither side is Hangul, then the segment break is removed.
	thisLastRuneEastAsianWidth := util.EastAsianWidth(thisLastRune)
	siblingFirstRuneEastAsianWidth := util.EastAsianWidth(siblingFirstRune)
	if (thisLastRuneEastAsianWidth == "F" ||
		thisLastRuneEastAsianWidth == "W" ||
		thisLastRuneEastAsianWidth == "H") &&
		(siblingFirstRuneEastAsianWidth == "F" ||
			siblingFirstRuneEastAsianWidth == "W" ||
			siblingFirstRuneEastAsianWidth == "H") {
		return unicode.Is(unicode.Hangul, thisLastRune) || unicode.Is(unicode.Hangul, siblingFirstRune)
	}

	// Rule3:
	//   Otherwise, if either the character before or after the segment break belongs to
	//   the space-discarding character set and it is a Unicode Punctuation (P*) or U+3000,
	//   then the segment break is removed.
	if util.IsSpaceDiscardingUnicodeRune(thisLastRune) ||
		unicode.IsPunct(thisLastRune) ||
		thisLastRune == '\u3000' ||
		util.IsSpaceDiscardingUnicodeRune(siblingFirstRune) ||
		unicode.IsPunct(siblingFirstRune) ||
		siblingFirstRune == '\u3000' {
		return false
	}

	// Rule4:
	//   Otherwise, the segment break is converted to a space (U+0020).
	return true
}

type withEastAsianLineBreaks struct {
	eastAsianLineBreaksStyle EastAsianLineBreaks
}

func (o *withEastAsianLineBreaks) SetConfig(c *renderer.Config) {
	c.Options[optEastAsianLineBreaks] = o.eastAsianLineBreaksStyle
}

func (o *withEastAsianLineBreaks) SetHTMLOption(c *Config) {
	c.EastAsianLineBreaks = o.eastAsianLineBreaksStyle
}

// WithEastAsianLineBreaks is a functional option that indicates whether softline breaks
// between east asian wide characters should be ignored.
func WithEastAsianLineBreaks(e EastAsianLineBreaks) interface {
	renderer.Option
	Option
} {
	return &withEastAsianLineBreaks{e}
}

// XHTML is an option name used in WithXHTML.
const optXHTML renderer.OptionName = "XHTML"

type withXHTML struct {
}

func (o *withXHTML) SetConfig(c *renderer.Config) {
	c.Options[optXHTML] = true
}

func (o *withXHTML) SetHTMLOption(c *Config) {
	c.XHTML = true
}

// WithXHTML is a functional option indicates that nodes should be rendered in
// xhtml instead of HTML5.
func WithXHTML() interface {
	Option
	renderer.Option
} {
	return &withXHTML{}
}

// Unsafe is an option name used in WithUnsafe.
const optUnsafe renderer.OptionName = "Unsafe"

type withUnsafe struct {
}

func (o *withUnsafe) SetConfig(c *renderer.Config) {
	c.Options[optUnsafe] = true
}

func (o *withUnsafe) SetHTMLOption(c *Config) {
	c.Unsafe = true
}

// WithUnsafe is a functional option that renders dangerous contents
// (raw htmls and potentially dangerous links) as it is.
func WithUnsafe() interface {
	renderer.Option
	Option
} {
	return &withUnsafe{}
}

// A Renderer struct is an implementation of renderer.NodeRenderer that renders
// nodes as (X)HTML.
type Renderer struct {
	count int
	dbg bool
	Config
}

// NewRenderer returns a new Renderer with given options.
func NewRenderer(dbg bool, opts ...Option) renderer.NodeRenderer {
//fmt.Println("dbg -- new renderer")
//	if dbg {log.Println("new renderer -- debugging!")}
	r := &Renderer{
		Config: NewConfig(),
	}
	r.dbg = dbg
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs .
func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// blocks
//fmt.Println("dbg -- reg funcs")

	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindTextBlock, r.renderTextBlock)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)

	// inlines

	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
}

func (r *Renderer) writeLines(w util.BufWriter, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		r.Writer.RawWrite(w, line.Value(source))
	}
}

// GlobalAttributeFilter defines attribute names which any elements can have.
var GlobalAttributeFilter = util.NewBytesFilter(
	[]byte("accesskey"),
	[]byte("autocapitalize"),
	[]byte("autofocus"),
	[]byte("class"),
	[]byte("contenteditable"),
	[]byte("dir"),
	[]byte("draggable"),
	[]byte("enterkeyhint"),
	[]byte("hidden"),
	[]byte("id"),
	[]byte("inert"),
	[]byte("inputmode"),
	[]byte("is"),
	[]byte("itemid"),
	[]byte("itemprop"),
	[]byte("itemref"),
	[]byte("itemscope"),
	[]byte("itemtype"),
	[]byte("lang"),
	[]byte("part"),
	[]byte("role"),
	[]byte("slot"),
	[]byte("spellcheck"),
	[]byte("style"),
	[]byte("tabindex"),
	[]byte("title"),
	[]byte("translate"),
)

func (r *Renderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// nothing to do
	node.SetAttributeString("el","mdDiv")
	if entering {
//fmt.Println("dbg -- start render Doc")
		r.count = 1
		docStr := `let mdDivObj = {
	typ:'div',
	id: 'mdDiv',
	style: {
		margin: '10px',
		border: '1px dashed blue',
		position: 'relative',
		minHeight: '200px',
	},
	};
	let mdDiv = azul.addElement(mdDivObj);
`
		_, _ = w.WriteString(docStr)

	} else {
//fmt.Println("dbg -- end render Doc")
	}
	return ast.WalkContinue, nil
}

// HeadingAttributeFilter defines attribute names which heading elements can have.
var HeadingAttributeFilter = GlobalAttributeFilter

func (r *Renderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		hdTyp := fmt.Sprintf("h%d",n.Level)
		hdStr := "let " + elNam + "= document.createElement('" + hdTyp + "');\n"
		_, _ = w.WriteString(hdStr)
		if n.Attributes() != nil {RenderElAttributes(w, n, HeadingAttributeFilter, elNam)}
	} else {
		pnode := n.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := n.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		hdStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(hdStr)
	}
	return ast.WalkContinue, nil
}

// BlockquoteAttributeFilter defines attribute names which blockquote elements can have.
var BlockquoteAttributeFilter = GlobalAttributeFilter.Extend(
	[]byte("cite"),
)

func (r *Renderer) renderBlockquote(
	w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		hdStr := "let " + elNam + "= document.createElement('blockquote');\n"
		_, _ = w.WriteString(hdStr)

		if node.Attributes() != nil {
			RenderElAttributes(w, node, BlockquoteAttributeFilter, elNam)
		}
	} else {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		hdStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(hdStr)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		elStr := "let " + elNam + "= document.createElement('pre');\n"
		_, _ = w.WriteString(elStr)
		r.count++
		el2Nam := fmt.Sprintf("el%d",r.count)
		el2Str := "let " + el2Nam + "= document.createElement('code');\n"
		_, _ = w.WriteString(el2Str)

		data := ""
		l := node.Lines().Len()
		for i := 0; i < l; i++ {
			line := node.Lines().At(i)
//		r.Writer.RawWrite(w, line.Value(source))
			data += string(line.Value(source))
		}
		el5Str := "let data='\n" + data + "'\n;"
		_, _ = w.WriteString(el5Str)
		r.count++
		el3Nam := fmt.Sprintf("el%d",r.count)
		el4Str := "let " + el3Nam + "= document.createTextNode(data);\n"
		_, _ = w.WriteString(el4Str)
		el6Str := el2Nam + ".appendChild(" + el3Nam + ");\n";
		_, _ = w.WriteString(el6Str)
		el7Str := elNam + ".appendChild(" + el2Nam + ");\n";
		_, _ = w.WriteString(el7Str)
//		r.writeLines(w, source, node)
	} else {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		hdStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(hdStr)
	}
	return ast.WalkContinue, nil
}

//requires work
func (r *Renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		_, _ = w.WriteString("<pre><code")
		language := n.Language(source)
		if language != nil {
			_, _ = w.WriteString(" class=\"language-")
			r.Writer.Write(w, language)
			_, _ = w.WriteString("\"")
		}
		_ = w.WriteByte('>')
		r.writeLines(w, source, n)
	} else {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
		_, _ = w.WriteString(dbgStr)
			hdStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(hdStr)
//		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderHTMLBlock(
	w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)
	if entering {
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		elStr := "let " + elNam + "= document.createElement('div');\n"
		_, _ = w.WriteString(elStr)
		if r.Unsafe {
			dataStr := ""
			l := n.Lines().Len()
			for i := 0; i < l; i++ {
				line := n.Lines().At(i)
//				r.Writer.SecureWrite(w, line.Value(source))
				dataStr += string(line.Value(source))
			}
			closure := n.ClosureLine
			dataStr += string(closure.Value(source))
			dat2Str := "let data='" + dataStr +"'"
			el2Str := elNam + ".innerhtml=" + dat2Str
			_, _ = w.WriteString(el2Str)
		} else {
			_, _ = w.WriteString("//<!-- raw HTML omitted -->\n")
		}
	} else {
		if n.HasClosure() {
			pnode := node.Parent()
			if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
			elNam, res := node.AttributeString("el")
			if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
			parElNam, res := pnode.AttributeString("el")
			if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}
			if r.dbg {
				dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
				_, _ = w.WriteString(dbgStr)
			}
			hdStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
			_, _ = w.WriteString(hdStr)
		}
	}
	return ast.WalkContinue, nil
}

// ListAttributeFilter defines attribute names which list elements can have.
var ListAttributeFilter = GlobalAttributeFilter.Extend(
	[]byte("start"),
	[]byte("reversed"),
	[]byte("type"),
)

func (r *Renderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.List)
	tag := "ul"
	if n.IsOrdered() {
		tag = "ol"
	}
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		elStr := "let " + elNam + "= document.createElement('" + tag + "');\n"
		_, _ = w.WriteString(elStr)

		if n.IsOrdered() && n.Start != 1 {
//			_, _ = fmt.Fprintf(w, " start=\"%d\"", n.Start)
			el2Str := elNam + fmt.Sprintf(".start='%d';\n", n.Start)
			_, _ = w.WriteString(el2Str)
		}
		if n.Attributes() != nil {RenderElAttributes(w, n, ListAttributeFilter, elNam)}
	} else {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		appStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(appStr)
	}
	return ast.WalkContinue, nil
}

// ListItemAttributeFilter defines attribute names which list item elements can have.
var ListItemAttributeFilter = GlobalAttributeFilter.Extend(
	[]byte("value"),
)

func (r *Renderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		elStr := "let " + elNam + "= document.createElement('li')\n;"
		_,_ = w.WriteString(elStr)
		if node.Attributes() != nil {RenderElAttributes(w, node, ListItemAttributeFilter, elNam)}
		fc := node.FirstChild()
		if fc != nil {
			if _, ok := fc.(*ast.TextBlock); !ok {
				_,_ = w.WriteString(elNam + ".textContent='\n';")
			}
		}
	} else {
//		_, _ = w.WriteString("</li>\n")
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}
		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		appStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(appStr)

	}
	return ast.WalkContinue, nil
}

// ParagraphAttributeFilter defines attribute names which paragraph elements can have.
var ParagraphAttributeFilter = GlobalAttributeFilter

func (r *Renderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		pStr:= "let " + elNam + "=document.createElement('p');\n"
		node.SetAttributeString("el",elNam)
		_, _ = w.WriteString(pStr)
		if node.Attributes() != nil {RenderElAttributes(w, node, ParagraphAttributeFilter, elNam)}
	} else {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)

//		_, _ = w.WriteString("</p>\n")
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTextBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		if n.NextSibling() != nil && n.FirstChild() != nil {
			_ = w.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

// ThematicAttributeFilter defines attribute names which hr elements can have.
var ThematicAttributeFilter = GlobalAttributeFilter.Extend(
	[]byte("align"),   // [Deprecated]
	[]byte("color"),   // [Not Standardized]
	[]byte("noshade"), // [Deprecated]
	[]byte("size"),    // [Deprecated]
	[]byte("width"),   // [Deprecated]
)

func (r *Renderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	//<hr>
	if !entering {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}
		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)
		return ast.WalkContinue, nil
	}
	// entering
	r.count++
	elNam := fmt.Sprintf("el%d",r.count)
	elStr:= "let " + elNam + "=document.createElement('hr');\n"
	node.SetAttributeString("el",elNam)
	_, _ = w.WriteString(elStr)

	if node.Attributes() != nil {
		RenderElAttributes(w, node, ThematicAttributeFilter, elNam)
	}
	return ast.WalkContinue, nil
}

// LinkAttributeFilter defines attribute names which link elements can have.
var LinkAttributeFilter = GlobalAttributeFilter.Extend(
	[]byte("download"),
	// []byte("href"),
	[]byte("hreflang"),
	[]byte("media"),
	[]byte("ping"),
	[]byte("referrerpolicy"),
	[]byte("rel"),
	[]byte("shape"),
	[]byte("target"),
)

func (r *Renderer) renderAutoLink(
	w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.AutoLink)
	if !entering {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)
		return ast.WalkContinue, nil
	}

// <a href="
	r.count++
	elNam := fmt.Sprintf("el%d",r.count)
	elStr:= "let " + elNam + "=document.createElement('a');\n"
	node.SetAttributeString("el",elNam)
	_, _ = w.WriteString(elStr)
	el2Str:= elNam + ".href=\""
	_, _ = w.WriteString(el2Str)

	url := n.URL(source)
//	label := n.Label(source)
	if n.AutoLinkType == ast.AutoLinkEmail && !bytes.HasPrefix(bytes.ToLower(url), []byte("mailto:")) {
		_, _ = w.WriteString("mailto:")
	}
	_, _ = w.Write(util.EscapeHTML(util.URLEscape(url, false)))
	_, _ = w.WriteString("\"\n;")

	if n.Attributes() != nil {
		RenderElAttributes(w, n, LinkAttributeFilter, elNam)
	}
	return ast.WalkContinue, nil
}

// CodeAttributeFilter defines attribute names which code elements can have.
var CodeAttributeFilter = GlobalAttributeFilter

func (r *Renderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		pStr:= "let " + elNam + "=document.createElement(\"code\");\n"
		node.SetAttributeString("el",elNam)
		_, _ = w.WriteString(pStr)
		if node.Attributes() != nil {
		//	_, _ = w.WriteString("<code")
			RenderElAttributes(w, node, CodeAttributeFilter, elNam)
		//	_ = w.WriteByte('>')
		}

		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			segment := c.(*ast.Text).Segment
			value := segment.Value(source)
			if bytes.HasSuffix(value, []byte("\n")) {
				r.Writer.RawWrite(w, value[:len(value)-1])
				r.Writer.RawWrite(w, []byte(" "))
			} else {
				r.Writer.RawWrite(w, value)
			}
		}
		return ast.WalkSkipChildren, nil
	}

//	_, _ = w.WriteString("</code>")
	pnode := node.Parent()
	if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
	elNam, res := node.AttributeString("el")
	if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
	parElNam, res := pnode.AttributeString("el")
	if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

	if r.dbg {
		dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
		_, _ = w.WriteString(dbgStr)
	}
	elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
	_, _ = w.WriteString(elStr)
	return ast.WalkContinue, nil
}

// EmphasisAttributeFilter defines attribute names which emphasis elements can have.
var EmphasisAttributeFilter = GlobalAttributeFilter

func (r *Renderer) renderEmphasis(
	w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	tag := "em"
	if n.Level == 2 {
		tag = "strong"
	}
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		node.SetAttributeString("el",elNam)
		elStr:= "let " + elNam + "=document.createElement('"+tag+"');\n"
		node.SetAttributeString("el",elNam)
		_, _ = w.WriteString(elStr)
		if n.Attributes() != nil {RenderElAttributes(w, node, EmphasisAttributeFilter, elNam)}
	} else {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}
		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)

//		_, _ = w.WriteString("</")
//		_, _ = w.WriteString(tag)
//		_ = w.WriteByte('>')
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		elStr:= "let " + elNam + "=document.createElement(\"a\");\n"
		node.SetAttributeString("el",elNam)
		_, _ = w.WriteString(elStr)
// need to add href ref
//		_, _ = w.WriteString("<a href=\"")
		if r.Unsafe || !IsDangerousURL(n.Destination) {
//			_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
			el2Str:= elNam + ".href=" + string(util.EscapeHTML(util.URLEscape(n.Destination, true)))
//		_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
			_, _ = w.WriteString(el2Str)
		}

//	el3Str := elNam + ".alt=\""
//	_, _ = w.WriteString(el3Str)
//	r.renderTexts(w, source, n)
//	_, _ = w.WriteString("\"\n")

		if n.Title != nil {
//		_, _ = w.WriteString(` title="`)
			el4Str := elNam + ".title='" + string(n.Title) + "'\n"
//		r.Writer.Write(w, n.Title)
			_,_ = w.WriteString(el4Str)
		}
		if n.Attributes() != nil {
			RenderElAttributes(w, n, LinkAttributeFilter, elNam)
		}
	} else {
//		_, _ = w.WriteString("</a>")
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}
		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)
	}
	return ast.WalkContinue, nil
}

// ImageAttributeFilter defines attribute names which image elements can have.
var ImageAttributeFilter = GlobalAttributeFilter.Extend(
	[]byte("align"),
	[]byte("border"),
	[]byte("crossorigin"),
	[]byte("decoding"),
	[]byte("height"),
	[]byte("importance"),
	[]byte("intrinsicsize"),
	[]byte("ismap"),
	[]byte("loading"),
	[]byte("referrerpolicy"),
	[]byte("sizes"),
	[]byte("srcset"),
	[]byte("usemap"),
	[]byte("width"),
)

func (r *Renderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)

		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	r.count++
	elNam := fmt.Sprintf("el%d",r.count)
	elStr:= "let " + elNam + "=document.createElement('img');\n"
	node.SetAttributeString("el",elNam)
	_, _ = w.WriteString(elStr)
	// need to add source
//	_, _ = w.WriteString("<img src=\"")
	if r.Unsafe || !IsDangerousURL(n.Destination) {
		el2Str:= elNam + ".src=" + string(util.EscapeHTML(util.URLEscape(n.Destination, true)))
//		_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
		_, _ = w.WriteString(el2Str)
	}
	el3Str := elNam + ".alt=\""
	_, _ = w.WriteString(el3Str)
	r.renderTexts(w, source, n)
	_, _ = w.WriteString("\"\n")


	if n.Title != nil {
//		_, _ = w.WriteString(` title="`)
		el4Str := elNam + ".title='" + string(n.Title) + "'\n"
//		r.Writer.Write(w, n.Title)
		_,_ = w.WriteString(el4Str)
	}

	if n.Attributes() != nil {
		RenderElAttributes(w, n, ImageAttributeFilter, elNam)
	}
	return ast.WalkSkipChildren, nil
}

func (r *Renderer) renderRawHTML(
	w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)
		return ast.WalkSkipChildren, nil
	}
		r.count++
		elNam := fmt.Sprintf("el%d",r.count)
		elStr:= "let " + elNam + "=document.createElement('div');\n"
		node.SetAttributeString("el",elNam)
		_, _ = w.WriteString(elStr)

	if r.Unsafe {
		n := node.(*ast.RawHTML)
		l := n.Segments.Len()
		//elnam.innerhtml =
		el2Str := elNam + ".innerhtml = `\n"
		_, _ = w.WriteString(el2Str)
		for i := 0; i < l; i++ {
			segment := n.Segments.At(i)
			_, _ = w.Write(segment.Value(source))
		}
		_, _ = w.WriteString("`\n")

		return ast.WalkSkipChildren, nil
	}
	_, _ = w.WriteString("<!-- raw HTML omitted -->")
	return ast.WalkSkipChildren, nil
}

func (r *Renderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		pnode := node.Parent()
		if pnode == nil {return ast.WalkStop, fmt.Errorf("no pnode")}
		elNam, res := node.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no el name!")}
		parElNam, res := pnode.AttributeString("el")
		if !res {return ast.WalkStop, fmt.Errorf("no parent el name!")}

		if r.dbg {
			dbgStr := fmt.Sprintf("// dbg -- el: %s parent:%s kind:%s\n", elNam, parElNam, pnode.Kind().String())
			_, _ = w.WriteString(dbgStr)
		}
		elStr := parElNam.(string) + ".appendChild(" + elNam.(string) + ");\n"
		_, _ = w.WriteString(elStr)

		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	segment := n.Segment

	r.count++
	elNam := fmt.Sprintf("el%d",r.count)
	n.SetAttributeString("el",elNam)


	if n.IsRaw() {
fmt.Println("dbg -- raw text")
//		_, _ = w.WriteString(segment.Value(source))

		r.Writer.RawWrite(w, segment.Value(source))
	} else {
		value := segment.Value(source)
		valStr := string(segment.Value(source))
//		txtStr := "let "+elNam+ "=document.createTextNode(\"" + string(value) + "\");\n"
//		_, _ = w.WriteString(txtStr)

//		r.Writer.Write(w, value)
//fmt.Printf("dbg -- text: %s\n", value)

		if n.HardLineBreak() || (n.SoftLineBreak() && r.HardWraps) {
				valStr += "\n"
//				_, _ = w.WriteString("<br>\n")
		} else if n.SoftLineBreak() {
			if r.EastAsianLineBreaks != EastAsianLineBreaksNone && len(valStr) != 0 {
				sibling := node.NextSibling()
				if sibling != nil && sibling.Kind() == ast.KindText {
					if siblingText := sibling.(*ast.Text).Value(source); len(siblingText) != 0 {
						thisLastRune := util.ToRune(value, len(value)-1)
						siblingFirstRune, _ := utf8.DecodeRune(siblingText)
						if r.EastAsianLineBreaks.softLineBreak(thisLastRune, siblingFirstRune) {
//							_ = w.WriteByte('\n')
							valStr += "\n"
						}
					}
				}
			} else {
				valStr += "\n"
//				_ = w.WriteByte('\n')
			}

		}
		datEl := elNam+"txt"
		datStr := "const " + datEl + "= `" + valStr + "`;\n"
		_, _ = w.WriteString(datStr)
		txtStr := "let "+elNam+ "=document.createTextNode(" + datEl + ");\n"
		_, _ = w.WriteString(txtStr)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	if r.dbg {fmt.Println("dbg -- string")}

	n := node.(*ast.String)
	if n.IsCode() {
		_, _ = w.Write(n.Value)
	} else {
		if n.IsRaw() {
			r.Writer.RawWrite(w, n.Value)
		} else {
			r.Writer.Write(w, n.Value)
		}
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTexts(w util.BufWriter, source []byte, n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if s, ok := c.(*ast.String); ok {
			_, _ = r.renderString(w, source, s, true)
		} else if t, ok := c.(*ast.Text); ok {
			_, _ = r.renderText(w, source, t, true)
		} else {
			r.renderTexts(w, source, c)
		}
	}
}

var dataPrefix = []byte("data-")

// RenderAttributes renders given node's attributes.
// You can specify attribute names to render by the filter.
// If filter is nil, RenderAttributes renders all attributes.
func RenderAttributes(w util.BufWriter, node ast.Node, filter util.BytesFilter) {
	for _, attr := range node.Attributes() {
		if filter != nil && !filter.Contains(attr.Name) {
			if !bytes.HasPrefix(attr.Name, dataPrefix) {
				continue
			}
		}
		_, _ = w.WriteString(" ")
		_, _ = w.Write(attr.Name)
		_, _ = w.WriteString(`="`)
		// TODO: convert numeric values to strings
		var value []byte
		switch typed := attr.Value.(type) {
		case []byte:
			value = typed
		case string:
			value = util.StringToReadOnlyBytes(typed)
		}
		_, _ = w.Write(util.EscapeHTML(value))
		_ = w.WriteByte('"')
	}
}

func RenderElAttributes(w util.BufWriter, node ast.Node, filter util.BytesFilter, elNam string) {
	for _, attr := range node.Attributes() {
		if filter != nil && !filter.Contains(attr.Name) {
			if !bytes.HasPrefix(attr.Name, dataPrefix) {
				continue
			}
		}
//		_, _ = w.Write(attr.Name)
//		_, _ = w.WriteString(`="`)
		// TODO: convert numeric values to strings
		var value string
		switch typed := attr.Value.(type) {
		case []byte:
			value = string(typed)
		case string:
			value = typed
		case int:
			value = fmt.Sprintf("%d",typed)
		//case float32
		}
		_, _ = w.WriteString(elNam + "." + string(attr.Name) + "='" + value + "';\n")
	}
}



// A Writer interface writes textual contents to a writer.
type Writer interface {
	// Write writes the given source to writer with resolving references and unescaping
	// backslash escaped characters.
	Write(writer util.BufWriter, source []byte)

	// RawWrite writes the given source to writer without resolving references and
	// unescaping backslash escaped characters.
	RawWrite(writer util.BufWriter, source []byte)

	// SecureWrite writes the given source to writer with replacing insecure characters.
	SecureWrite(writer util.BufWriter, source []byte)
}

var replacementCharacter = []byte("\ufffd")

// A WriterConfig struct has configurations for the HTML based writers.
type WriterConfig struct {
	// EscapedSpace is an option that indicates that a '\' escaped half-space(0x20) should not be rendered.
	EscapedSpace bool
}

// A WriterOption interface sets options for HTML based writers.
type WriterOption func(*WriterConfig)

// WithEscapedSpace is a WriterOption indicates that a '\' escaped half-space(0x20) should not be rendered.
func WithEscapedSpace() WriterOption {
	return func(c *WriterConfig) {
		c.EscapedSpace = true
	}
}

type defaultWriter struct {
	WriterConfig
}

// NewWriter returns a new Writer.
func NewWriter(opts ...WriterOption) Writer {
	w := &defaultWriter{}
	for _, opt := range opts {
		opt(&w.WriterConfig)
	}
	return w
}

func escapeRune(writer util.BufWriter, r rune) {
	if r < 256 {
		v := util.EscapeHTMLByte(byte(r))
		if v != nil {
			_, _ = writer.Write(v)
			return
		}
	}
	_, _ = writer.WriteRune(util.ToValidRune(r))
}

func (d *defaultWriter) SecureWrite(writer util.BufWriter, source []byte) {
	n := 0
	l := len(source)
	for i := 0; i < l; i++ {
		if source[i] == '\u0000' {
			_, _ = writer.Write(source[i-n : i])
			n = 0
			_, _ = writer.Write(replacementCharacter)
			continue
		}
		n++
	}
	if n != 0 {
		_, _ = writer.Write(source[l-n:])
	}
}

func (d *defaultWriter) RawWrite(writer util.BufWriter, source []byte) {
	n := 0
	l := len(source)
	for i := 0; i < l; i++ {
		v := util.EscapeHTMLByte(source[i])
		if v != nil {
			_, _ = writer.Write(source[i-n : i])
			n = 0
			_, _ = writer.Write(v)
			continue
		}
		n++
	}
	if n != 0 {
		_, _ = writer.Write(source[l-n:])
	}
}

func (d *defaultWriter) Write(writer util.BufWriter, source []byte) {
	escaped := false
	var ok bool
	limit := len(source)
	n := 0
	for i := 0; i < limit; i++ {
		c := source[i]
		if escaped {
			if util.IsPunct(c) {
				d.RawWrite(writer, source[n:i-1])
				n = i
				escaped = false
				continue
			}
			if d.EscapedSpace && c == ' ' {
				d.RawWrite(writer, source[n:i-1])
				n = i + 1
				escaped = false
				continue
			}
		}
		if c == '\x00' {
			d.RawWrite(writer, source[n:i])
			d.RawWrite(writer, replacementCharacter)
			n = i + 1
			escaped = false
			continue
		}
		if c == '&' {
			pos := i
			next := i + 1
			if next < limit && source[next] == '#' {
				nnext := next + 1
				if nnext < limit {
					nc := source[nnext]
					// code point like #x22;
					if nnext < limit && nc == 'x' || nc == 'X' {
						start := nnext + 1
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsHexDecimal)
						if ok && i < limit && source[i] == ';' && i-start < 7 {
							v, _ := strconv.ParseUint(util.BytesToReadOnlyString(source[start:i]), 16, 32)
							d.RawWrite(writer, source[n:pos])
							n = i + 1
							escapeRune(writer, rune(v))
							continue
						}
						// code point like #1234;
					} else if nc >= '0' && nc <= '9' {
						start := nnext
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsNumeric)
						if ok && i < limit && i-start < 8 && source[i] == ';' {
							v, _ := strconv.ParseUint(util.BytesToReadOnlyString(source[start:i]), 10, 32)
							d.RawWrite(writer, source[n:pos])
							n = i + 1
							escapeRune(writer, rune(v))
							continue
						}
					}
				}
			} else {
				start := next
				i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsAlphaNumeric)
				// entity reference
				if ok && i < limit && source[i] == ';' {
					name := util.BytesToReadOnlyString(source[start:i])
					entity, ok := util.LookUpHTML5EntityByName(name)
					if ok {
						d.RawWrite(writer, source[n:pos])
						n = i + 1
						d.RawWrite(writer, entity.Characters)
						continue
					}
				}
			}
			i = next - 1
		}
		if c == '\\' {
			escaped = true
			continue
		}
		escaped = false
	}
	d.RawWrite(writer, source[n:])
}

// DefaultWriter is a default instance of the Writer.
var DefaultWriter = NewWriter()

var bDataImage = []byte("data:image/")
var bPng = []byte("png;")
var bGif = []byte("gif;")
var bJpeg = []byte("jpeg;")
var bWebp = []byte("webp;")
var bSvg = []byte("svg+xml;")
var bJs = []byte("javascript:")
var bVb = []byte("vbscript:")
var bFile = []byte("file:")
var bData = []byte("data:")

func hasPrefix(s, prefix []byte) bool {
	return len(s) >= len(prefix) && bytes.Equal(bytes.ToLower(s[0:len(prefix)]), bytes.ToLower(prefix))
}

// IsDangerousURL returns true if the given url seems a potentially dangerous url,
// otherwise false.
func IsDangerousURL(url []byte) bool {
	if hasPrefix(url, bDataImage) && len(url) >= 11 {
		v := url[11:]
		if hasPrefix(v, bPng) || hasPrefix(v, bGif) ||
			hasPrefix(v, bJpeg) || hasPrefix(v, bWebp) ||
			hasPrefix(v, bSvg) {
			return false
		}
		return true
	}
	return hasPrefix(url, bJs) || hasPrefix(url, bVb) ||
		hasPrefix(url, bFile) || hasPrefix(url, bData)
}

func GetRenderer(dbg bool) (r renderer.Renderer) {
	if dbg {log.Println("*** debugging ***")}
	r = renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(NewRenderer(dbg), 1000)))
	return r
}
