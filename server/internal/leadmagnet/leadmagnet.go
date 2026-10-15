// Package leadmagnet holds the files the bot hands out to people who subscribe to the
// channel (see internal/api/lead_magnet.go for the flow). The files are embedded so their
// content is pinned to the release, and published at FetchURL so Telegram can download them.
package leadmagnet

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
)

//go:embed files/ucimo-top100-slov.pdf
var top100 []byte

// filesBase is where the files in files/ are served from. The repository is public, and Telegram
// can reach GitHub (it cannot reach this site, and uploading the bytes through the Cloudflare
// relay the bot's API calls use corrupted them). Telegram fetches a file once; after that the
// bot sends the file_id it got back.
const filesBase = "https://raw.githubusercontent.com/llladno/serbian-language-app/main/server/internal/leadmagnet/files/"

// Magnet is one downloadable file.
type Magnet struct {
	Slug     string // the id used in the deep link: t.me/<bot>?start=lm_<slug>[_<source>]
	FetchURL string // where Telegram downloads the file from; its last part is the name the person sees
	Data     []byte
	SHA      string // sha256 of Data, to know when a cached Telegram file_id is stale
}

var registry = map[string]Magnet{}

func init() {
	add(Magnet{Slug: "top100", FetchURL: filesBase + "ucimo-top100-slov.pdf", Data: top100})
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
