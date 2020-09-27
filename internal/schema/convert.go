package schema

import (
	"time"

	"github.com/dghubble/go-twitter/twitter"
)

func (t *Tweet) Populate(o *twitter.Tweet) error {
	t.Id = o.ID
	createdAt, err := time.ParseInLocation(time.RubyDate, o.CreatedAt, time.UTC)
	if err != nil {
		return err
	}
	t.CreatedAt = createdAt.Format(time.RFC3339)
	if o.Coordinates != nil {
		t.Coordinates = &Coordinates{}
		t.Coordinates.Populate(o.Coordinates)
	}
	t.Lang = o.Lang
	t.PossibleSensitive = o.PossiblySensitive
	t.Stats = &TweetStats{}
	t.Stats.Populate(o)
	t.Witheld = &WitheldInfo{}
	t.Witheld.Populate(o)

	if o.InReplyToStatusID != 0 || o.InReplyToUserID != 0 {
		t.Reply = &ReplyInfo{
			ScreenName: o.InReplyToScreenName,
			UserId:     o.InReplyToUserID,
			StatusId:   o.InReplyToStatusID,
		}
	}

	if o.RetweetedStatus != nil {
		t.Retweet = &Tweet{}
		t.Retweet.Populate(o.RetweetedStatus)
	}

	if o.Truncated {
		t.Text = o.ExtendedTweet.FullText
	} else {
		t.Text = o.Text
	}

	if o.QuotedStatus != nil {
		t.QuotedStatus = &Tweet{}
		t.QuotedStatus.Populate(o.QuotedStatus)
	}

	if o.Entities != nil || o.ExtendedEntities != nil {
		t.Entities = &Entities{}
		t.Entities.Populate(o)
	}

	return nil
}

func (c *Coordinates) Populate(o *twitter.Coordinates) {
	c.Lat = o.Coordinates[0]
	c.Long = o.Coordinates[1]
	c.Type = o.Type
}

func (t *TweetStats) Populate(o *twitter.Tweet) {
	t.QuoteCount = int32(o.QuoteCount)
	t.ReplyCount = int32(o.ReplyCount)
	t.RetweetCount = int32(o.RetweetCount)
	t.Retweeted = o.Retweeted
}

func (w *WitheldInfo) Populate(t *twitter.Tweet) {
	w.WithheldCopyright = t.WithheldCopyright
	w.WithheldInCountries = t.WithheldInCountries
	w.WithheldScope = t.WithheldScope
}

func (e *Entities) Populate(o *twitter.Tweet) {
	if o.Entities.Hashtags != nil {
		for _, h := range o.Entities.Hashtags {
			e.Hashtags = append(e.Hashtags, &Hashtag{
				Text:    h.Text,
				Indices: toSchemaIndices(h.Indices),
			})
		}
	}

	if o.Entities.UserMentions != nil {
		for _, u := range o.Entities.UserMentions {
			e.Mentions = append(e.Mentions, &Mention{
				Id:         u.ID,
				Name:       u.Name,
				ScreenName: u.ScreenName,
			})
		}
	}

	if o.Entities.Urls != nil {
		for _, u := range o.Entities.Urls {
			e.Urls = append(e.Urls, &URLInfo{
				Indices:     toSchemaIndices(u.Indices),
				DisplayUrl:  u.DisplayURL,
				ExpandedUrl: u.ExpandedURL,
				Url:         u.URL,
			})
		}
	}

	if o.ExtendedEntities != nil {
		for _, m := range o.ExtendedEntities.Media {
			md := &Media{}
			md.Populate(m)
			e.Media = append(e.Media, md)
		}
	}
}

func (m *Media) Populate(o twitter.MediaEntity) {
	m.MediaUrl = o.MediaURL
	m.MediaUrlHttps = o.MediaURLHttps
	m.Id = o.ID
	m.SourceStatusId = o.SourceStatusID
	m.Type = o.Type
	m.Size = &MediaSizes{}
	m.Size.Populate(o.Sizes)

	if o.VideoInfo.DurationMillis > 0 {
		m.VideoInfo = &VideoInfo{}
		m.VideoInfo.Populate(o.VideoInfo)
	}
}

func (m *MediaSizes) Populate(o twitter.MediaSizes) {
	m.Thumb = toSchemaMediaSize(o.Thumb)
	m.Large = toSchemaMediaSize(o.Large)
	m.Medium = toSchemaMediaSize(o.Medium)
	m.Small = toSchemaMediaSize(o.Small)
}

func (vi *VideoInfo) Populate(o twitter.VideoInfo) {
	vi.AspectRatio = &AspectRatio{
		Width:  int32(o.AspectRatio[0]),
		Height: int32(o.AspectRatio[1]),
	}
	vi.DurationMillis = int32(o.DurationMillis)
	for _, v := range o.Variants {
		vi.Variants = append(vi.Variants, &VideoVariant{
			ContentType: v.ContentType,
			Url:         v.URL,
			Bitrate:     int32(v.Bitrate),
		})
	}
}

func toSchemaMediaSize(o twitter.MediaSize) *MediaSize {
	return &MediaSize{
		Height: int32(o.Height),
		Width:  int32(o.Width),
		Resize: o.Resize,
	}
}

func toSchemaIndices(h twitter.Indices) *Indices {
	return &Indices{
		Start: int32(h.Start()),
		End:   int32(h.End()),
	}
}
