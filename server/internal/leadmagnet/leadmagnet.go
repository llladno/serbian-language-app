// Package leadmagnet holds the files the bot hands out to people who subscribe to the
// channel (see internal/api/lead_magnet.go for the flow). The files are embedded, so a
// deploy ships them and nothing outside the binary can serve them.
package leadmagnet

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
)

//go:embed files/top100.pdf
var top100 []byte

// Magnet is one downloadable file.
type Magnet struct {
	Slug     string // the id used in the deep link: t.me/<bot>?start=lm_<slug>[_<source>]
	Filename string // the name the person sees in Telegram when the file is uploaded
	URLName  string // the last part of the download link: Telegram names a fetched file after it
	Data     []byte
	SHA      string // sha256 of Data, to know when a cached Telegram file_id is stale
}

var registry = map[string]Magnet{}

func init() {
	add(Magnet{Slug: "top100", Filename: "Топ-100 сербских слов — ucimo.pdf", URLName: "ucimo-top100-slov.pdf", Data: top100})
}

func add(m Magnet) {
	sum := sha256.Sum256(m.Data)
	m.SHA = hex.EncodeToString(sum[:])
	registry[m.Slug] = m
}

// Get returns the magnet with the given slug.
func Get(slug string) (Magnet, bool) {
	m, ok := registry[slug]
	return m, ok
}
