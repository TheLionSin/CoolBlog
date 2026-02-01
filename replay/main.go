package main

import (
	"context"
	"flag"
	"go_blog/config"
	"go_blog/internal/repositories"
	"log"
)

func main() {
	limit := flag.Int("limit", 50, "how many DEAD events to reset to NEW")
	flag.Parse()

	config.ConnectDB()
	repo := repositories.NewOutboxRepository(config.DB)

	ctx := context.Background()

	items, err := repo.FetchDead(ctx, *limit)
	if err != nil {
		log.Fatal("fetch dead:", err)
	}
	if len(items) == 0 {
		log.Println("no dead events")
		return
	}

	for _, it := range items {
		if err := repo.ResetDeadToNew(ctx, it.ID); err != nil {
			log.Printf("reset failed id=%d event_id=%s err=%v", it.ID, it.EventID, err)
			continue
		}
		log.Printf("reset DEAD→NEW id=%d event_id=%s", it.ID, it.EventID)
	}
}
