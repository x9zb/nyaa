package model

import (
	"context"
	"fmt"
	"html/template"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	elastic "gopkg.in/olivere/elastic.v5"

	"net/url"

	"github.com/NyaaPantsu/nyaa/config"
	"github.com/NyaaPantsu/nyaa/util"
	"github.com/bradfitz/slice"
)

const (
	// TorrentStatusNormal Int for Torrent status normal
	TorrentStatusNormal = 1
	// TorrentStatusRemake Int for Torrent status remake
	TorrentStatusRemake = 2
	// TorrentStatusTrusted Int for Torrent status trusted
	TorrentStatusTrusted = 3
	// TorrentStatusAPlus Int for Torrent status a+
	TorrentStatusAPlus = 4
	// TorrentStatusBlocked Int for Torrent status locked
	TorrentStatusBlocked = 5
)

// Feed struct
type Feed struct {
	ID        int
	Name      string
	Hash      string
	Magnet    string
	Timestamp string
}

// Torrent model
type Torrent struct {
	ID          uint      `gorm:"column:torrent_id;primary_key"`
	Name        string    `gorm:"column:torrent_name"`
	Hash        string    `gorm:"column:torrent_hash"`
	Category    int       `gorm:"column:category"`
	SubCategory int       `gorm:"column:sub_category"`
	Status      int       `gorm:"column:status"`
	Hidden      bool      `gorm:"column:hidden"`
	Date        time.Time `gorm:"column:date"`
	UploaderID  uint      `gorm:"column:uploader"`
	Downloads   int       `gorm:"column:downloads"`
	Stardom     int       `gorm:"column:stardom"`
	Filesize    int64     `gorm:"column:filesize"`
	Description string    `gorm:"column:description"`
	WebsiteLink string    `gorm:"column:website_link"`
	Trackers    string    `gorm:"column:trackers"`
	DeletedAt   *time.Time

	Uploader    *User        `gorm:"AssociationForeignKey:UploaderID;ForeignKey:user_id"`
	OldUploader string       `gorm:"-"` // ???????
	OldComments []OldComment `gorm:"ForeignKey:torrent_id"`
	Comments    []Comment    `gorm:"ForeignKey:torrent_id"`

	Seeders    uint32    `gorm:"column:seeders"`
	Leechers   uint32    `gorm:"column:leechers"`
	Completed  uint32    `gorm:"column:completed"`
	LastScrape time.Time `gorm:"column:last_scrape"`
	FileList   []File    `gorm:"ForeignKey:torrent_id"`
}

// Size : Returns the total size of memory recursively allocated for this struct
// FIXME: Is it deprecated?
func (t Torrent) Size() (s int) {
	s = int(reflect.TypeOf(t).Size())
	return

}

// TableName : Return the name of torrents table
func (t Torrent) TableName() string {
	return config.TorrentsTableName
}

// Identifier : Return the identifier of a torrent
func (t *Torrent) Identifier() string {
	return "torrent_" + strconv.Itoa(int(t.ID))
}

// IsNormal : Return if a torrent status is normal
func (t Torrent) IsNormal() bool {
	return t.Status == TorrentStatusNormal
}

// IsRemake : Return if a torrent status is normal
func (t Torrent) IsRemake() bool {
	return t.Status == TorrentStatusRemake
}

// IsTrusted : Return if a torrent status is trusted
func (t Torrent) IsTrusted() bool {
	return t.Status == TorrentStatusTrusted
}

// IsAPlus : Return if a torrent status is a+
func (t Torrent) IsAPlus() bool {
	return t.Status == TorrentStatusAPlus
}

// IsBlocked : Return if a torrent status is locked
func (t *Torrent) IsBlocked() bool {
	return t.Status == TorrentStatusBlocked
}

// IsDeleted : Return if a torrent status is deleted
func (t *Torrent) IsDeleted() bool {
	return t.DeletedAt != nil
}

// AddToESIndex : Adds a torrent to Elastic Search
func (t Torrent) AddToESIndex(client *elastic.Client) error {
	ctx := context.Background()
	torrentJSON := t.ToJSON()
	_, err := client.Index().
		Index(config.DefaultElasticsearchIndex).
		Type(config.DefaultElasticsearchType).
		Id(strconv.FormatUint(uint64(torrentJSON.ID), 10)).
		BodyJson(torrentJSON).
		Refresh("true").
		Do(ctx)
	return err
}

// DeleteFromESIndex : Removes a torrent from Elastic Search
func (t Torrent) DeleteFromESIndex(client *elastic.Client) error {
	ctx := context.Background()
	_, err := client.Delete().
		Index(config.DefaultElasticsearchIndex).
		Type(config.DefaultElasticsearchType).
		Id(strconv.FormatInt(int64(t.ID), 10)).
		Do(ctx)
	return err
}

// ParseTrackers : Takes an array of trackers, adds needed trackers and parse it to url string
func (t *Torrent) ParseTrackers(trackers []string) {
	v := url.Values{}
	if len(config.NeededTrackers) > 0 { // if we have some needed trackers configured
		if len(trackers) == 0 {
			trackers = config.Trackers
		} else {
			for _, id := range config.NeededTrackers {
				found := false
				for _, tracker := range trackers {
					if tracker == config.Trackers[id] {
						found = true
						break
					}
				}
				if !found {
					trackers = append(trackers, config.Trackers[id])
				}
			}
		}
	}
	v["tr"] = trackers
	t.Trackers = v.Encode()
}

// GetTrackersArray : Convert trackers string to Array
func (t *Torrent) GetTrackersArray() (trackers []string) {
	v, _ := url.ParseQuery(t.Trackers)
	trackers = v["tr"]
	return
}

/* We need a JSON object instead of a Gorm structure because magnet URLs are
   not in the database and have to be generated dynamically */

// APIResultJSON for torrents in json for api
type APIResultJSON struct {
	Torrents         []TorrentJSON `json:"torrents"`
	QueryRecordCount int           `json:"queryRecordCount"`
	TotalRecordCount int           `json:"totalRecordCount"`
}

// CommentJSON for comment model in json
type CommentJSON struct {
	Username   string        `json:"username"`
	UserID     int           `json:"user_id"`
	UserAvatar string        `json:"user_avatar"`
	Content    template.HTML `json:"content"`
	Date       time.Time     `json:"date"`
}

// FileJSON for file model in json
type FileJSON struct {
	Path     string `json:"path"`
	Filesize int64  `json:"filesize"`
}

// TorrentJSON for torrent model in json for api
type TorrentJSON struct {
	ID           uint          `json:"id"`
	Name         string        `json:"name"`
	Status       int           `json:"status"`
	Hash         string        `json:"hash"`
	Date         string        `json:"date"`
	Filesize     int64         `json:"filesize"`
	Description  template.HTML `json:"description"`
	Comments     []CommentJSON `json:"comments"`
	SubCategory  string        `json:"sub_category"`
	Category     string        `json:"category"`
	Downloads    int           `json:"downloads"`
	UploaderID   uint          `json:"uploader_id"`
	UploaderName template.HTML `json:"uploader_name"`
	OldUploader  template.HTML `json:"uploader_old"`
	WebsiteLink  template.URL  `json:"website_link"`
	Magnet       template.URL  `json:"magnet"`
	TorrentLink  template.URL  `json:"torrent"`
	Seeders      uint32        `json:"seeders"`
	Leechers     uint32        `json:"leechers"`
	Completed    uint32        `json:"completed"`
	LastScrape   time.Time     `json:"last_scrape"`
	FileList     []FileJSON    `json:"file_list"`
}

// ToJSON converts a model.Torrent to its equivalent JSON structure
func (t *Torrent) ToJSON() TorrentJSON {
	var trackers []string
	if t.Trackers == "" {
		trackers = config.Trackers
	} else {
		trackers = t.GetTrackersArray()
	}
	magnet := util.InfoHashToMagnet(strings.TrimSpace(t.Hash), t.Name, trackers...)
	commentsJSON := make([]CommentJSON, 0, len(t.OldComments)+len(t.Comments))
	for _, c := range t.OldComments {
		commentsJSON = append(commentsJSON, CommentJSON{Username: c.Username, UserID: -1, Content: template.HTML(c.Content), Date: c.Date.UTC()})
	}
	for _, c := range t.Comments {
		if c.User != nil {
			commentsJSON = append(commentsJSON, CommentJSON{Username: c.User.Username, UserID: int(c.User.ID), Content: util.MarkdownToHTML(c.Content), Date: c.CreatedAt.UTC(), UserAvatar: c.User.MD5})
		} else {
			commentsJSON = append(commentsJSON, CommentJSON{})
		}
	}

	// Sort comments by date
	slice.Sort(commentsJSON, func(i, j int) bool {
		return commentsJSON[i].Date.Before(commentsJSON[j].Date)
	})

	fileListJSON := make([]FileJSON, 0, len(t.FileList))
	for _, f := range t.FileList {
		fileListJSON = append(fileListJSON, FileJSON{
			Path:     filepath.Join(f.Path()...),
			Filesize: f.Filesize,
		})
	}

	// Sort file list by lowercase filename
	slice.Sort(fileListJSON, func(i, j int) bool {
		return strings.ToLower(fileListJSON[i].Path) < strings.ToLower(fileListJSON[j].Path)
	})

	uploader := ""
	var uploaderID uint
	if t.Hidden {
		uploader = "れんちょん"
		uploaderID = 0
	} else if t.Uploader != nil {
		uploader = t.Uploader.Username
		uploaderID = t.UploaderID
	}
	torrentlink := ""
	if t.ID <= config.LastOldTorrentID && len(config.TorrentCacheLink) > 0 {
		if config.IsSukebei() {
			torrentlink = "" // torrent cache doesn't have sukebei torrents
		} else {
			torrentlink = fmt.Sprintf(config.TorrentCacheLink, t.Hash)
		}
	} else if t.ID > config.LastOldTorrentID && len(config.TorrentStorageLink) > 0 {
		torrentlink = fmt.Sprintf(config.TorrentStorageLink, t.Hash)
	}
	res := TorrentJSON{
		ID:           t.ID,
		Name:         t.Name,
		Status:       t.Status,
		Hash:         t.Hash,
		Date:         t.Date.Format(time.RFC3339),
		Filesize:     t.Filesize,
		Description:  util.MarkdownToHTML(t.Description),
		Comments:     commentsJSON,
		SubCategory:  strconv.Itoa(t.SubCategory),
		Category:     strconv.Itoa(t.Category),
		Downloads:    t.Downloads,
		UploaderID:   uploaderID,
		UploaderName: util.SafeText(uploader),
		OldUploader:  util.SafeText(t.OldUploader),
		WebsiteLink:  util.Safe(t.WebsiteLink),
		Magnet:       template.URL(magnet),
		TorrentLink:  util.Safe(torrentlink),
		Leechers:     t.Leechers,
		Seeders:      t.Seeders,
		Completed:    t.Completed,
		LastScrape:   t.LastScrape,
		FileList:     fileListJSON,
	}

	return res
}

/* Complete the functions when necessary... */

// TorrentsToJSON : Map Torrents to TorrentsToJSON without reallocations
func TorrentsToJSON(t []Torrent) []TorrentJSON {
	json := make([]TorrentJSON, len(t))
	for i := range t {
		json[i] = t[i].ToJSON()
	}
	return json
}
