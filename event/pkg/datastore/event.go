package datastore

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"

	"github.com/dizaknz/gkgraph/event/pkg/types"
)

var ErrEventLinkNotFound = errors.New("Event link not found")

type EventDatastore struct {
	db     bolt.Conn
	logger log.Logger
}

func NewEventDatastore(db bolt.Conn, logger log.Logger) *EventDatastore {
	return &EventDatastore{
		db:     db,
		logger: logger,
	}
}

func (ds *EventDatastore) Add(ev *types.Event) error {
	tx, err := ds.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	cypher := []string{}
	props := map[string]interface{}{}
	cypher = append(cypher, fmt.Sprintf("CREATE (n:%s { id: {id}, timestamp: {timestamp}", ev.Type))
	props["id"] = ev.ID
	props["timestamp"] = ev.Timestamp.Unix()

	for _, av := range ev.Attributes {
		cypher = append(cypher, fmt.Sprintf(", %s: {%s}", av.Name, av.Name))
		props[av.Name] = av.Value
	}
	cypher = append(cypher, "})")
	res, err := ds.db.ExecNeo(strings.Join(cypher, " "), props)
	if err != nil {
		level.Error(ds.logger).Log("Failed to create event", "err", err)
		return err
	}
	num, _ := res.RowsAffected()
	if num != 1 {
		level.Error(ds.logger).Log("No event created", "ID", ev.ID)
		return fmt.Errorf("event: %s could not be created", ev.ID)
	}
	if len(ev.Links) > 0 {
		queries := make([]string, len(ev.Links))
		params := make([]map[string]interface{}, len(ev.Links))
		for i, link := range ev.Links {
			lattrs := []string{}
			param := map[string]interface{}{
				"id":     ev.ID,
				"linkID": link.EventID,
			}
			if len(link.Attributes) > 0 {
				lattrs = append(lattrs, "{")
				comma := ""
				for _, av := range link.Attributes {
					lattrs = append(lattrs, fmt.Sprintf("%s %s: {%s}", comma, av.Name, av.Name))
					param[av.Name] = av.Value
					comma = ","
				}
				lattrs = append(lattrs, "}")
			}

			queries[i] = fmt.Sprintf(
				`MATCH (e1:%s), (e2:%s) WHERE e1.id = {id} AND e2.id = {linkID} CREATE (e1)-[:%s%s]->(e2)`,
				ev.Type,
				link.EventType,
				link.LinkType.String(),
				strings.Join(lattrs, " "),
			)
			params[i] = param
		}
		pipeline, err := ds.db.PreparePipeline(queries...)
		defer pipeline.Close()
		if err != nil {
			level.Error(ds.logger).Log("msg", "Failed to prepare pipeline", "err", err)
			return err
		}
		results, err := pipeline.ExecPipeline(params...)
		if err != nil {
			level.Error(ds.logger).Log("msg", "Failed to create event links", "err", err)
			return err
		}
		for i, res := range results {
			num, err := res.RowsAffected()
			if err != nil {
				if num == -1 {
					level.Error(ds.logger).Log(
						"msg", "Event link could not be created",
						"event_id", params[i]["id"],
						"event_link_id", params[i]["linkID"],
						"err", ErrEventLinkNotFound,
					)
					return ErrEventLinkNotFound
				}
				level.Error(ds.logger).Log("msg", "Failed to create event link", "err", err.Error())
				return err
			}
			level.Debug(ds.logger).Log(
				"msg", "Created event links",
				"link_id", ev.Links[i].EventID,
				"num", num,
			)
		}
	}
	tx.Commit()
	return nil
}
