package rss

import (
	"encoding/xml"
	"fmt"
	"time"
)

const (
	ItunesNS = "http://www.itunes.com/dtds/podcast-1.0.dtd"
)

type Feed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Itunes  string   `xml:"xmlns:itunes,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Language    string     `xml:"language,omitempty"`
	Author      string     `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author,omitempty"`
	Image       *ItunesImg `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image,omitempty"`
	Category    *Category  `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category,omitempty"`
	Explicit    string     `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd explicit,omitempty"`
	Items       []*Item    `xml:"item"`
}

type ItunesImg struct {
	XMLName xml.Name `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Href    string   `xml:"href,attr"`
}

type Category struct {
	XMLName xml.Name `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
	Text    string   `xml:"text,attr"`
}

type Item struct {
	Title       string     `xml:"title"`
	Description string     `xml:"description,omitempty"`
	Enclosure   Enclosure  `xml:"enclosure"`
	GUID        string     `xml:"guid"`
	PubDate     string     `xml:"pubDate"`
	Duration    string     `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration,omitempty"`
	Image       *ItunesImg `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image,omitempty"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func NewFeed(title, description, link string) *Feed {
	return &Feed{
		Version: "2.0",
		Itunes:  ItunesNS,
		Channel: Channel{
			Title:       title,
			Description: description,
			Link:        link,
			Language:    "ru",
			Explicit:    "no",
		},
	}
}

func (f *Feed) SetImage(url string) {
	f.Channel.Image = &ItunesImg{Href: url}
}

func (f *Feed) ClearImage() {
	f.Channel.Image = nil
}

func (f *Feed) AddItem(item *Item) {
	f.Channel.Items = append([]*Item{item}, f.Channel.Items...)
}

func (f *Feed) RemoveItem(guid string) bool {
	for i, item := range f.Channel.Items {
		if item.GUID == guid {
			f.Channel.Items = append(f.Channel.Items[:i], f.Channel.Items[i+1:]...)
			return true
		}
	}
	return false
}

func (f *Feed) FindItem(guid string) *Item {
	for _, item := range f.Channel.Items {
		if item.GUID == guid {
			return item
		}
	}
	return nil
}

func NewItem(title, audioURL string, fileSize int64, duration string, pubDate time.Time) *Item {
	return &Item{
		Title: title,
		Enclosure: Enclosure{
			URL:    audioURL,
			Length: fileSize,
			Type:   "audio/mpeg",
		},
		GUID:     audioURL,
		PubDate:  pubDate.UTC().Format(time.RFC1123Z),
		Duration: duration,
	}
}

func Marshal(f *Feed) ([]byte, error) {
	data, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling RSS: %w", err)
	}
	return append([]byte(xml.Header), data...), nil
}

func Unmarshal(data []byte) (*Feed, error) {
	var f Feed
	if err := xml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("unmarshaling RSS: %w", err)
	}
	// Go xml loses xmlns:itunes on roundtrip — restore it
	f.Version = "2.0"
	f.Itunes = ItunesNS
	return &f, nil
}
