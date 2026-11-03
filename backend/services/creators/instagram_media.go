package creators

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MediaScanLimit is how many recent posts one scan reads.
//
// Business discovery returns recent media, never a full history, so no
// limit here would buy a complete picture — raising it only widens the
// window a little. 25 is what the edge returns by default and is enough
// to tell a creator who discloses brand work from one who does not.
const MediaScanLimit = 25

// Post is one of a creator's recent posts, as far as this needs it.
type Post struct {
	Permalink string
	Caption   string
	// PostedAt is zero where the timestamp did not parse, which is not
	// worth discarding the post over — the caption is the evidence and
	// the permalink proves where it came from.
	PostedAt time.Time
}

type mediaResponse struct {
	BusinessDiscovery struct {
		Username       string `json:"username"`
		Name           string `json:"name"`
		FollowersCount int64  `json:"followers_count"`
		MediaCount     int64  `json:"media_count"`
		Media          struct {
			Data []struct {
				Caption   string `json:"caption"`
				Permalink string `json:"permalink"`
				Timestamp string `json:"timestamp"`
			} `json:"data"`
		} `json:"media"`
	} `json:"business_discovery"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// LookupMedia reads a creator's recent posts along with their profile.
//
// Separate from Lookup on purpose. Pulling someone's media is a bigger
// ask than reading their follower count, so it is something a caller
// has to choose rather than something every sync does by default. The
// profile comes back with it because the identity guard has to run on
// the same response — attributing another account's posts to a real
// named person is the failure worth preventing here.
func (i *Instagram) LookupMedia(ctx context.Context, handle string, limit int) (Profile, []Post, error) {
	if !i.cfg.complete() {
		return Profile{}, nil, ErrInstagramNotConfigured
	}
	handle = cleanHandle(handle)
	if handle == "" {
		return Profile{}, nil, fmt.Errorf("no handle to look up")
	}
	if limit <= 0 || limit > MediaScanLimit {
		limit = MediaScanLimit
	}

	body, err := i.fetch(ctx, fmt.Sprintf(
		"business_discovery.username(%s){username,name,followers_count,media_count,"+
			"media.limit(%d){caption,permalink,timestamp}}", handle, limit))
	if err != nil {
		return Profile{}, nil, err
	}
	var out mediaResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return Profile{}, nil, fmt.Errorf("instagram: malformed response: %w", err)
	}
	if out.Error != nil {
		return Profile{}, nil, fmt.Errorf("instagram: %s (code %d)", out.Error.Message, out.Error.Code)
	}
	d := out.BusinessDiscovery
	if d.Username == "" {
		return Profile{}, nil, fmt.Errorf("instagram: no public professional account for %q", handle)
	}

	posts := make([]Post, 0, len(d.Media.Data))
	for _, m := range d.Media.Data {
		if m.Permalink == "" {
			// Without a permalink there is no evidence to show, and a
			// detection nobody can check is the thing this avoids.
			continue
		}
		p := Post{Permalink: m.Permalink, Caption: m.Caption}
		p.PostedAt = parseIGTime(m.Timestamp)
		posts = append(posts, p)
	}
	return Profile{
		Username:   d.Username,
		Name:       d.Name,
		Followers:  d.FollowersCount,
		MediaCount: d.MediaCount,
	}, posts, nil
}

// parseIGTime reads the timestamp business discovery returns.
//
// Instagram sends "2026-03-11T09:30:00+0000" — ISO 8601 with no colon
// in the offset, which time.RFC3339 refuses. Parsing with RFC3339
// alone fails on every real post and returns the zero time quietly, so
// the offset-without-colon layout is tried first and RFC3339 second,
// in case the format is ever tidied up.
func parseIGTime(s string) time.Time {
	for _, layout := range []string{"2006-01-02T15:04:05-0700", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
