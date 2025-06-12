package web

import (
	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/rweb"
)

// ----- OTHER PAGE -----

// otherPageHandler demonstrates an alternative page construction technique
func otherPageHandler(c rweb.Context) error {
	return c.WriteHTML(otherHTMLPage())
}

func otherHTMLPage() (out string) {
	b := element.NewBuilder()
	b.HtmlPage("body {background-color:#eee;}", "<title>My Other Page</title>", otherBody{})
	return b.String()
}

// Define the otherBody component
type otherBody struct{}

func (ob otherBody) Render(b *element.Builder) (x any) {
	b.H1().T("This is my other page")

	b.P().R(
		b.T("This is a simple example of using the Element library to generate HTML."),
	)
	b.Input("type", "text").R(
	// b.Span().T("I shouldn't be here"),
	)

	b.DivClass("footer").T("About | Privacy | Logout")
	return
}
