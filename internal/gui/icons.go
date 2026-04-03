//go:build gui

package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ── Material Design icon resources ────────────────────────────────────────────

// mdIcon builds a themed fill-based SVG resource from one or more Material
// Design path strings. theme.NewThemedResource replaces fill="#000000" with
// the current theme foreground colour, adapting to light and dark mode.
func mdIcon(name string, paths ...string) fyne.Resource {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">`)
	for _, p := range paths {
		fmt.Fprintf(&b, `<path fill="#000000" d="%s"/>`, p)
	}
	b.WriteString(`</svg>`)
	return theme.NewThemedResource(fyne.NewStaticResource(name, []byte(b.String())))
}

var (
	iconSong         = mdIcon("song.svg", "M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z")
	iconHeader       = mdIcon("header.svg", "M5 4v3h5.5v12h3V7H19V4z")
	iconMedia        = mdIcon("media.svg", "M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z")
	iconPresentation = mdIcon("presentation.svg", "M10 8v8l5-4-5-4z", "M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z")
	iconImage        = mdIcon("image.svg", "M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z")
	iconItem         = mdIcon("item.svg", "M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z")
	iconAttachment   = mdIcon("attachment.svg", "M16.5 6v11.5c0 2.21-1.79 4-4 4s-4-1.79-4-4V5c0-1.38 1.12-2.5 2.5-2.5s2.5 1.12 2.5 2.5v10.5c0 .55-.45 1-1 1s-1-.45-1-1V6H10v9.5c0 1.38 1.12 2.5 2.5 2.5s2.5-1.12 2.5-2.5V5c0-2.21-1.79-4-4-4S7 2.79 7 5v12.5c0 3.04 2.46 5.5 5.5 5.5s5.5-2.46 5.5-5.5V6h-1.5z")
)

// itemTypeIcon returns the icon for a PCO item type.
func itemTypeIcon(itemType string) fyne.Resource {
	switch itemType {
	case "song":
		return iconSong
	case "header":
		return iconHeader
	case "media":
		return iconMedia
	case "presentation":
		return iconPresentation
	case "image":
		return iconImage
	default:
		return iconItem
	}
}
