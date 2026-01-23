package edu

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
)

func Init(ctx context.Context) error {
	log.Info("Initializing Educational Extension...")

	e := db.GetEngine(ctx)
	if e == nil {
		log.Fatal("Educational Extension: Database engine not available")
		return nil
	}
	
	if err := e.Sync(new(Assignment), new(Submission), new(TestResult), new(UserRole)); err != nil {
		log.Error("Educational Extension: Failed to sync database schema: %v", err)
		return err
	}

	repo := NewRepository(e)

	RegisterNotifier(repo)

	log.Info("Educational Extension initialized successfully.")
	return nil
}
