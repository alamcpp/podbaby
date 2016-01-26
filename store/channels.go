package store

import (
	"fmt"
	"github.com/danjac/podbaby/models"
	"github.com/jmoiron/sqlx"
	"strings"
)

const maxRecommendations = 20

type ChannelReader interface {
	SelectAll(DataHandler) ([]models.Channel, error)
	SelectByCategoryID(DataHandler, int64) ([]models.Channel, error)
	SelectSubscribed(DataHandler, int64) ([]models.Channel, error)
	SelectRelated(DataHandler, int64) ([]models.Channel, error)
	SelectRecommended(DataHandler) ([]models.Channel, error)
	SelectRecommendedByUserID(DataHandler, int64) ([]models.Channel, error)
	Search(DataHandler, string) ([]models.Channel, error)
	GetByID(DataHandler, int64) (*models.Channel, error)
	GetByURL(DataHandler, string) (*models.Channel, error)
}

type ChannelWriter interface {
	Create(DataHandler, *models.Channel) error
	AddCategories(DataHandler, *models.Channel) error
	AddPodcasts(DataHandler, *models.Channel) error
}

type ChannelStore interface {
	ChannelReader
	ChannelWriter
}

type channelSqlStore struct {
	ChannelReader
	ChannelWriter
}

func newChannelStore() ChannelStore {
	return &channelSqlStore{
		ChannelReader: &channelSqlReader{},
		ChannelWriter: &channelSqlWriter{},
	}
}

type channelSqlReader struct{}

func (r *channelSqlReader) SelectAll(dh DataHandler) ([]models.Channel, error) {
	q := "SELECT id, title, description, url, image, website FROM channels"
	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q)
}

func (r *channelSqlReader) SelectByCategoryID(dh DataHandler, categoryID int64) ([]models.Channel, error) {
	q := `
    SELECT c.id, c.title, c.image, c.description, c.website, c.url
    FROM channels c
    JOIN channels_categories cc 
    ON cc.channel_id = c.id
    WHERE cc.category_id=$1
    GROUP BY c.id
    ORDER BY c.title`
	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q, categoryID)
}

func (r *channelSqlReader) SelectRelated(dh DataHandler, channelID int64) ([]models.Channel, error) {
	q := `
    SELECT c.id, c.title, c.image, c.description, c.website, c.url
    FROM channels c
    JOIN subscriptions s ON s.channel_id=c.id
    WHERE s.user_id in (
      SELECT user_id FROM subscriptions WHERE channel_id=$1
    ) AND s.channel_id != $1
    GROUP BY c.id
    ORDER BY RANDOM() DESC LIMIT 3`

	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q, channelID)
}

func (r *channelSqlReader) SelectRecommended(dh DataHandler) ([]models.Channel, error) {
	q := `
    SELECT c.id, c.title, c.image, c.description, c.website, c.url
    FROM channels c
    JOIN subscriptions s ON s.channel_id = c.id
    GROUP BY c.id
    ORDER BY COUNT(DISTINCT(s.id)) DESC LIMIT $1
    `
	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q, maxRecommendations)
}

func (r *channelSqlReader) SelectRecommendedByUserID(dh DataHandler, userID int64) ([]models.Channel, error) {
	q := `
    WITH user_subs AS (SELECT channel_id FROM subscriptions WHERE user_id=$1)
    SELECT c.id, c.title, c.description, c.image, c.url, c.website
    FROM channels c
    JOIN channels_categories cc ON cc.channel_id=c.id
    WHERE cc.category_id IN (
       SELECT cc.category_id FROM channels_categories cc
       WHERE cc.channel_id IN (SELECT channel_id FROM user_subs)
    )
    AND c.id NOT IN (SELECT channel_id FROM user_subs)
    GROUP BY c.id
    ORDER BY RANDOM()
    LIMIT $2`
	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q, userID, maxRecommendations)
}

func (r *channelSqlReader) SelectSubscribed(dh DataHandler, userID int64) ([]models.Channel, error) {

	q := `
    SELECT c.id, c.title, c.description, c.image, c.url, c.website
    FROM channels c
    JOIN subscriptions s ON s.channel_id = c.id
    WHERE s.user_id=$1 AND title IS NOT NULL AND title != ''
    GROUP BY c.id
    ORDER BY title`
	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q, userID)
}

func (r *channelSqlReader) Search(dh DataHandler, query string) ([]models.Channel, error) {

	q := `
    SELECT c.id, c.title, c.description, c.url, c.image, c.website
    FROM channels c, plainto_tsquery($1) as q
    WHERE (c.tsv @@ q)
    ORDER BY ts_rank_cd(c.tsv, plainto_tsquery($1)) DESC LIMIT 20`
	var channels []models.Channel
	return channels, sqlx.Select(dh, &channels, q, query)
}

func (r *channelSqlReader) GetByURL(dh DataHandler, url string) (*models.Channel, error) {
	q := `
    SELECT id, title, description, url, image, website
    FROM channels
    WHERE url=$1`
	channel := &models.Channel{}
	return channel, sqlx.Get(dh, channel, q, url)
}

func (r *channelSqlReader) GetByID(dh DataHandler, id int64) (*models.Channel, error) {
	q := `
    SELECT c.id, c.title, c.description, c.url, c.image, c.website
    FROM channels c
    WHERE id=$1`
	channel := &models.Channel{}
	return channel, sqlx.Get(dh, channel, q, id)
}

type channelSqlWriter struct{}

func (w *channelSqlWriter) Create(dh DataHandler, ch *models.Channel) error {

	q := `SELECT upsert_channel (
    :url, 
    :title, 
    :description, 
    :image, 
    :keywords, 
    :website
    )`

	q, args, err := sqlx.Named(q, ch)
	if err != nil {
		return err
	}

	return dh.QueryRowx(dh.Rebind(q), args...).Scan(&ch.ID)
}

func (w *channelSqlWriter) AddCategories(dh DataHandler, channel *models.Channel) error {
	if len(channel.Categories) == 0 {
		return nil
	}
	args := []interface{}{
		channel.ID,
	}

	params := make([]string, 0, len(channel.Categories))
	for i, category := range channel.Categories {
		params = append(params, fmt.Sprintf("$%v", i+2))
		args = append(args, category)
	}

	q := fmt.Sprintf("SELECT add_categories($1, ARRAY[%s])", strings.Join(params, ", "))
	_, err := dh.Exec(q, args...)
	return err
}

func (w *channelSqlWriter) AddPodcasts(dh DataHandler, channel *models.Channel) error {

	q := `SELECT insert_podcast(
        :channel_id, 
        :guid,
        :title, 
        :description, 
        :enclosure_url, 
        :source,
        :pub_date)`

	stmt, err := dh.PrepareNamed(dh.Rebind(q))
	if err != nil {
		return err
	}

	for _, pc := range channel.Podcasts {
		pc.ChannelID = channel.ID
		err = stmt.QueryRowx(&pc).Scan(&pc.ID)
		if err != nil {
			return err
		}
	}
	return nil

}