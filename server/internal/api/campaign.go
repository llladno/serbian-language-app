package api

import (
	"crypto/subtle"
	"fmt"
	"html"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/campaign"
	"github.com/grisha/serbian-app/server/internal/mail"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
)

// The «return» campaign (docs/superpowers/specs/2026-11-08-return-campaign-design.md): a one-off message to learners who never
// opened lesson 02, in three waves. Nothing is sent by itself. The owner opens a wave through the service call below (a dry run
// first); opening takes people off the frozen audience and gives each a button style, and RunCampaignSweep then sends their
// messages through the day, each inside the person's own daytime.

// The campaign writes only between 10:00 and 20:00 Belgrade time, whatever the learner's own timezone, and spreads a wave over that
// window with the weight on the evening: each reminder sweep (main.go runs one every campaignSweepEvery) sends only its share of
// what is still due today, and a sweep's share grows through the day (campaignWeight), so a wave leaves as a trickle that thickens
// towards the evening and never as a burst.
const (
	campaignFromHour   = 10
	campaignToHour     = 20
	campaignSweepEvery = 20 * time.Minute
	// campaignEmailsPerSweepMax is a ceiling on one sweep's e-mails whatever the share works out to.
	campaignEmailsPerSweepMax = 30
)

// campaignZone is the clock of the campaign's window.
var campaignZone = func() *time.Location {
	if loc, err := time.LoadLocation("Europe/Belgrade"); err == nil {
		return loc
	}
	return time.FixedZone("CET", 3600)
}()

// campaignWeight is how heavy a sweep is at this clock time in Belgrade: 1 in the morning (10:00-14:00), 2 in the afternoon (14:00-17:00)
// and 3 in the evening (17:00-20:00), 0 outside the window. A day's messages therefore leave about a fifth in the morning, a third
// in the afternoon and nearly half in the evening.
func campaignWeight(t time.Time) int {
	switch h := t.In(campaignZone).Hour(); {
	case h < campaignFromHour || h >= campaignToHour:
		return 0
	case h < 14:
		return 1
	case h < 17:
		return 2
	}
	return 3
}

// campaignWeightsFrom is the weight of the sweep at now and of every later sweep today, one every campaignSweepEvery until the window
// closes, so a sweep can take its share of what is left: now's weight over this total. Both are 0 outside the window.
func campaignWeightsFrom(now time.Time) (this, total int) {
	this = campaignWeight(now)
	if this == 0 {
		return 0, 0
	}
	local := now.In(campaignZone)
	end := time.Date(local.Year(), local.Month(), local.Day(), campaignToHour, 0, 0, 0, campaignZone)
	for t := now; t.Before(end); t = t.Add(campaignSweepEvery) {
		total += campaignWeight(t)
	}
	return this, total
}

// ceilDiv is a/b rounded up (b > 0).
func ceilDiv(a, b int) int { return (a + b - 1) / b }

// campaignAuth lets through a request with the campaign's bearer token. Without a token configured the endpoints do not exist.
func (h handlers) campaignAuth(w http.ResponseWriter, r *http.Request) bool {
	token := h.Config.CampaignServiceToken
	if token == "" {
		fail(w, http.StatusNotFound, "not found")
		return false
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
		fail(w, http.StatusUnauthorized, "bad token")
		return false
	}
	return true
}

// campaignChannelPlan is what opening a wave does (or, in a dry run, would do) in one channel.
type campaignChannelPlan struct {
	Channel  string      `json:"channel"`
	Taken    int         `json:"taken"`
	Segments map[int]int `json:"segments"`
	Styles   map[int]int `json:"styles"`
	Leader   int         `json:"leader"`
	Share    int         `json:"leader_share_permille"`
}

// postCampaignWave handles POST /api/service/campaigns/return/wave {wave, dry_run, force}. Waves open in order, one at a time. The
// first one freezes the audience (everybody who qualifies at that moment; 15% of them become the group that is never written to).
// A later wave refuses while the earlier one still has messages to go out, unless force is set. A dry run changes nothing.
func (h handlers) postCampaignWave(w http.ResponseWriter, r *http.Request) {
	if !h.campaignAuth(w, r) {
		return
	}
	var req struct {
		Wave   int  `json:"wave"`
		DryRun bool `json:"dry_run"`
		Force  bool `json:"force"`
	}
	if err := decode(r, &req); err != nil || req.Wave < 1 || req.Wave > campaign.Waves {
		fail(w, http.StatusBadRequest, "wave must be 1, 2 or 3")
		return
	}
	opened, err := h.Store.CampaignWavesOpened(campaign.Name)
	if err != nil {
		log.Printf("campaign wave: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	if req.Wave != opened+1 {
		fail(w, http.StatusConflict, fmt.Sprintf("waves open in order: the next one is %d", opened+1))
		return
	}
	if req.Wave > 1 && !req.Force {
		unsent, err := h.Store.CampaignUnsent(campaign.Name)
		if err != nil {
			log.Printf("campaign wave: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		if unsent > 0 {
			fail(w, http.StatusConflict, fmt.Sprintf("%d messages of the earlier wave have not gone out yet (send force to open anyway)", unsent))
			return
		}
	}
	now := h.Now()

	frozen, err := h.Store.CampaignFrozen(campaign.Name)
	if err != nil {
		log.Printf("campaign wave: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	var audience []store.CampaignCandidate
	if !frozen {
		if audience, err = h.Store.CampaignAudience(campaign.Name, now); err != nil {
			log.Printf("campaign wave: %v", err)
			fail(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !req.DryRun {
			rows := make([]store.CampaignRow, 0, len(audience))
			for _, c := range audience {
				kind := store.CampaignSent
				if campaign.Holdout(c.UserID) {
					kind = store.CampaignHoldout
				}
				rows = append(rows, store.CampaignRow{UserID: c.UserID, Channel: c.Channel, Segment: c.Segment, Kind: kind, Ord: campaign.Ord(c.UserID)})
			}
			if err := h.Store.CampaignFreeze(campaign.Name, rows, now); err != nil {
				log.Printf("campaign wave: %v", err)
				fail(w, http.StatusInternalServerError, "internal error")
				return
			}
		}
	}

	var plans []campaignChannelPlan
	for _, channel := range []string{store.ChannelTelegram, store.ChannelEmail} {
		leader, share := campaign.StyleDirect, campaign.LeaderShare(req.Wave)
		if req.Wave > 1 {
			stats, err := h.Store.CampaignStyleStats(campaign.Name, channel)
			if err != nil {
				log.Printf("campaign wave: %v", err)
				fail(w, http.StatusInternalServerError, "internal error")
				return
			}
			var perStyle [campaign.NumStyles]campaign.Stat
			for style, s := range stats {
				if style >= 0 && style < campaign.NumStyles {
					perStyle[style] = campaign.Stat{Sent: s.Sent, Clicked: s.Clicked}
				}
			}
			leader = campaign.Leader(perStyle)
		}
		shares := campaign.Shares(req.Wave, leader)
		styleOf := func(id string) int { return campaign.StyleFor(id, req.Wave, shares) }

		var taken []store.CampaignTaken
		if frozen || !req.DryRun {
			if taken, err = h.Store.CampaignTake(campaign.Name, channel, req.Wave, campaign.WaveSize(req.Wave), styleOf, !req.DryRun); err != nil {
				log.Printf("campaign wave: %v", err)
				fail(w, http.StatusInternalServerError, "internal error")
				return
			}
		} else {
			taken = dryTake(audience, channel, req.Wave, styleOf)
		}
		plan := campaignChannelPlan{Channel: channel, Taken: len(taken), Segments: map[int]int{}, Styles: map[int]int{}, Leader: leader, Share: share}
		for _, t := range taken {
			plan.Segments[t.Segment]++
			plan.Styles[t.Style]++
		}
		plans = append(plans, plan)
		if !req.DryRun {
			if err := h.Store.CampaignRecordWave(campaign.Name, req.Wave, channel, leader, share, now); err != nil {
				log.Printf("campaign wave: %v", err)
			}
		}
	}

	holdout, total := 0, 0
	for _, c := range audience {
		total++
		if campaign.Holdout(c.UserID) {
			holdout++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"wave": req.Wave, "dry_run": req.DryRun, "audience_frozen_now": !frozen && !req.DryRun,
		"audience_total": total, "holdout": holdout, "channels": plans,
	})
}

// dryTake is what a first wave would take from an audience that has not been frozen, for a dry run.
func dryTake(audience []store.CampaignCandidate, channel string, wave int, styleOf func(string) int) []store.CampaignTaken {
	var pool []store.CampaignCandidate
	for _, c := range audience {
		if c.Channel == channel && !campaign.Holdout(c.UserID) {
			pool = append(pool, c)
		}
	}
	sort.Slice(pool, func(i, j int) bool { return campaign.Ord(pool[i].UserID) < campaign.Ord(pool[j].UserID) })
	if n := campaign.WaveSize(wave); n >= 0 && len(pool) > n {
		pool = pool[:n]
	}
	out := make([]store.CampaignTaken, 0, len(pool))
	for _, c := range pool {
		out = append(out, store.CampaignTaken{UserID: c.UserID, Segment: c.Segment, Style: styleOf(c.UserID)})
	}
	return out
}

// getCampaignResults handles GET /api/service/campaigns/return/results: per wave, channel, segment and style how many people were
// taken, how many messages went out and how many buttons were pressed; how each wave was opened; and, per segment, how many of the
// people who got a message and how many of the group that did not reached the lesson after theirs.
func (h handlers) getCampaignResults(w http.ResponseWriter, r *http.Request) {
	if !h.campaignAuth(w, r) {
		return
	}
	res, err := h.Store.CampaignResults(campaign.Name)
	if err != nil {
		log.Printf("campaign results: %v", err)
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	type cell struct {
		Wave    int    `json:"wave"`
		Channel string `json:"channel"`
		Segment int    `json:"segment"`
		Style   int    `json:"style"`
		Taken   int    `json:"taken"`
		Sent    int    `json:"sent"`
		Clicked int    `json:"clicked"`
	}
	type group struct {
		Segment int    `json:"segment"`
		Kind    string `json:"kind"`
		People  int    `json:"people"`
		Reached int    `json:"reached"`
	}
	type wave struct {
		Wave        int    `json:"wave"`
		Channel     string `json:"channel"`
		Leader      int    `json:"leader"`
		LeaderShare int    `json:"leader_share_permille"`
		OpenedAt    string `json:"opened_at"`
	}
	out := struct {
		Cells  []cell  `json:"cells"`
		Waves  []wave  `json:"waves"`
		Groups []group `json:"groups"`
	}{Cells: []cell{}, Waves: []wave{}, Groups: []group{}}
	for _, c := range res.Cells {
		out.Cells = append(out.Cells, cell{c.Wave, c.Channel, c.Segment, c.Style, c.Taken, c.Sent, c.Clicked})
	}
	for _, x := range res.Waves {
		out.Waves = append(out.Waves, wave{x.Wave, x.Channel, x.Leader, x.LeaderShare, x.OpenedAt})
	}
	for _, g := range res.Groups {
		out.Groups = append(out.Groups, group{g.Segment, g.Kind, g.People, g.Reached})
	}
	writeJSON(w, http.StatusOK, out)
}

// RunCampaignSweep sends the messages of the waves that have been opened, between 10:00 and 20:00 Belgrade time, each sweep sending
// only its share of what is still due that day, heavier towards the evening (see campaignWeight). Called from the reminder ticker; a person who already had a
// study message today waits for a later sweep. Someone who can no longer be reached is skipped. It does nothing unless
// CAMPAIGN_RETURN is on.
func RunCampaignSweep(deps Deps, now time.Time) error {
	if !deps.Config.Campaign {
		return nil
	}
	h := handlers{deps}
	due, err := deps.Store.CampaignDue(campaign.Name)
	if err != nil {
		return fmt.Errorf("campaign sweep: %w", err)
	}
	weight, weightsLeft := campaignWeightsFrom(now)
	if weight == 0 || weightsLeft == 0 {
		return nil // outside 10:00-20:00 Belgrade: everything waits for the next morning
	}
	// This sweep's share of what is due in each channel: spread over the sweeps that are left today.
	var dueTG, dueMail int
	for _, row := range due {
		if row.Channel == store.ChannelEmail {
			dueMail++
		} else {
			dueTG++
		}
	}
	budgetTG := ceilDiv(dueTG*weight, weightsLeft)
	budgetMail := ceilDiv(dueMail*weight, weightsLeft)
	if budgetMail > campaignEmailsPerSweepMax {
		budgetMail = campaignEmailsPerSweepMax
	}
	sentTG, sentMail := 0, 0
	today := now.UTC().Format("2006-01-02")
	for _, row := range due {
		if (row.Channel == store.ChannelEmail && sentMail >= budgetMail) || (row.Channel != store.ChannelEmail && sentTG >= budgetTG) {
			continue // this sweep has sent its share of the channel
		}
		prefs, err := deps.Store.BotPrefs(row.UserID)
		if err != nil {
			log.Printf("campaign sweep: prefs for %s: %v", row.UserID, err)
			continue
		}
		state, err := deps.Store.BotReminderState(row.UserID)
		if err != nil {
			log.Printf("campaign sweep: state for %s: %v", row.UserID, err)
			continue
		}
		nudge, err := deps.Store.RepeatNudgeInfo(row.UserID)
		if err != nil {
			log.Printf("campaign sweep: nudge info for %s: %v", row.UserID, err)
			continue
		}
		if state.AllDoneDate == today || now.Sub(state.InactiveSentAt) < studyMessageGap || now.Sub(nudge.LastNudge) < studyMessageGap {
			continue // one study message a day: this one waits
		}

		msg := campaign.For(row.Segment)
		switch row.Channel {
		case store.ChannelTelegram:
			chat, ok := deps.Store.TelegramChatID(row.UserID)
			if !ok || !prefs.Learning {
				h.skipCampaign(row)
				continue
			}
			if !h.claimCampaign(row, now) {
				continue
			}
			sentTG++
			button := &telegram.InlineButton{Label: msg.Buttons[row.Style], CampaignURL: campaign.LessonURL(deps.Config.AppBaseURL, row.Segment, row.Token)}
			deps.EnqueueTelegramPhoto(chat, campaign.PhotoURL(deps.Config.CampaignPhotoBase, row.Segment), campaign.TelegramHTML(msg.Telegram), button, telegram.PriorityNormal)
			// The campaign message is this quiet spell's «come back» message: the generic reminder keeps off the same day.
			if err := deps.Store.SetBotReminderInactiveSentAt(row.UserID, now); err != nil {
				log.Printf("campaign sweep: save inactivity state for %s: %v", row.UserID, err)
			}
		case store.ChannelEmail:
			to, ok := deps.Store.VerifiedEmail(row.UserID)
			if out, err := deps.Store.EmailOptedOut(row.UserID); err != nil || out || !ok {
				if err != nil {
					log.Printf("campaign sweep: opt-out for %s: %v", row.UserID, err)
					continue
				}
				h.skipCampaign(row)
				continue
			}
			if !h.claimCampaign(row, now) {
				continue
			}
			base := strings.TrimRight(deps.Config.AppBaseURL, "/")
			subject, text, body := mail.RenderCampaign(mail.CampaignMail{
				Subject: msg.Subject, Preheader: msg.Preheader, Heading: msg.Heading, Lead: msg.Lead, Box: msg.Box,
				BtnLabel: msg.Buttons[row.Style], Image: campaign.ImageURL(base, row.Segment),
				Link:        campaign.LessonURL(base, row.Segment, row.Token),
				Unsubscribe: base + "/api/unsubscribe?t=" + row.Token,
			})
			deps.Async(func() { deps.SendMail(to, subject, text, body) })
			sentMail++
		default:
			continue
		}
		// The swallow's tour was put off for everybody by migration 039; this message promises it, so bring it back.
		if err := deps.Store.ResetSystemOnboarding(row.UserID); err != nil {
			log.Printf("campaign sweep: reset onboarding for %s: %v", row.UserID, err)
		}
	}
	return nil
}

// claimCampaign takes a campaign message for sending and reports whether this sweep got it. It is claimed before anything is sent, so
// two sweeps running at once never send the same person's message twice.
func (h handlers) claimCampaign(row store.CampaignDueRow, now time.Time) bool {
	ok, err := h.Store.CampaignClaim(row.ID, now)
	if err != nil {
		log.Printf("campaign sweep: claim %d: %v", row.ID, err)
		return false
	}
	return ok
}

func (h handlers) skipCampaign(row store.CampaignDueRow) {
	if err := h.Store.CampaignSkip(row.ID); err != nil {
		log.Printf("campaign sweep: skip %d: %v", row.ID, err)
	}
}

// unsubscribePage handles GET /api/unsubscribe?t=<token of a campaign message>: a page that asks for confirmation. Nothing
// changes on GET, so a mail scanner that opens every link cannot unsubscribe anybody.
func (h handlers) unsubscribePage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	if _, ok, err := h.Store.CampaignUserByToken(token); err != nil || !ok {
		writeUnsubscribeHTML(w, http.StatusOK, "Ссылка устарела", "Эта ссылка больше не работает. Если письма всё ещё приходят, напишите нам из приложения.", "")
		return
	}
	writeUnsubscribeHTML(w, http.StatusOK, "Отписаться от писем об учёбе?",
		"Мы перестанем присылать напоминания об учёбе на почту и в Telegram. Вы сможете вернуться в любой момент.",
		`<form method="post" action="/api/unsubscribe?t=`+html.EscapeString(token)+`"><button type="submit">Отписаться</button></form>`)
}

// unsubscribe handles POST /api/unsubscribe?t=<token>: the person asked not to get study messages. It records the e-mail opt-out and
// switches off the study reminders in Telegram too, so they do not keep arriving there.
func (h handlers) unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	id, ok, err := h.Store.CampaignUserByToken(token)
	if err != nil {
		log.Printf("unsubscribe: %v", err)
		writeUnsubscribeHTML(w, http.StatusInternalServerError, "Не получилось", "Попробуйте ещё раз чуть позже.", "")
		return
	}
	if !ok {
		writeUnsubscribeHTML(w, http.StatusOK, "Ссылка устарела", "Эта ссылка больше не работает. Если письма всё ещё приходят, напишите нам из приложения.", "")
		return
	}
	now := h.Now()
	prefs, err := h.Store.BotPrefs(id)
	if err == nil {
		prefs.Learning = false
		err = h.Store.SetBotPrefs(id, prefs, now)
	}
	if err == nil {
		err = h.Store.SetEmailOptout(id, now)
	}
	if err != nil {
		log.Printf("unsubscribe: %v", err)
		writeUnsubscribeHTML(w, http.StatusInternalServerError, "Не получилось", "Попробуйте ещё раз чуть позже.", "")
		return
	}
	writeUnsubscribeHTML(w, http.StatusOK, "Вы отписались", "Больше не будем писать про учёбу. Hvala (спасибо), что были с нами.", "")
}

func writeUnsubscribeHTML(w http.ResponseWriter, status int, title, text, extra string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">`+
		`<title>%s</title></head><body style="font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;max-width:420px;margin:15vh auto;padding:0 16px;text-align:center;color:#172326">`+
		`<h1 style="font-size:22px">%s</h1><p style="color:#546063;line-height:1.5">%s</p>%s</body></html>`,
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(text), extra)
}
